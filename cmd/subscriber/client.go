package subscriber

import (
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/adamsanghera/go-websub/internal/discovery"
)

/*
	According to https://www.w3.org/TR/websub/#subscriber, a Subscriber
	is a service that discovers hubs and topics.
	According to https://www.w3.org/TR/websub/#conformance-classes, a Subscriber
	MUST:
	- support specific content-delivery mechanisms
	- send subscription requests according to the spec
	- acknowledge content-delivery requests with a HTTP 2xx code
	MAY:
	- request specific lease durations, related to the subscription
	- include a secret in the sub request.  If it does, then it
		- MUST use the secret to verify the signature in the content delivery request
	- request that a subscription be deactivated with an unsubscribe mechanism
*/

// Client subscribes to topic hubs, following the websub protocol
type Client struct {
	port         string
	callback     string
	topicsToHubs map[string]map[string]struct{}
	topicsToSelf map[string]string

	pendingSubs   map[string]struct{} // perhaps point to a boolean sticky/not-sticky?
	pendingUnSubs map[string]struct{}
	activeSubs    map[string]struct{}
}

// NewClient creates and returns a new subscription client
// Callback needs to be formatted like http{s}://website.domain:{port}/endpoint
func NewClient(callback string, port string) *Client {
	// Create the client
	c := &Client{
		callback:     callback,
		topicsToHubs: make(map[string]map[string]struct{}),
		topicsToSelf: make(map[string]string),

		subscribedTopicURLs: make(map[string]map[string]struct{}),
	}

	// Register the callback, and listen/serve
	http.HandleFunc(callback, c.Callback)
	go func() {
		if err := http.ListenAndServe(":"+port, nil); err != nil {
			panic(err)
		}
	}()

	return c
}

// GetHubsForTopic returns all hubs associated with a given topic
func (sc *Client) GetHubsForTopic(topic string) []string {
	hubs := make([]string, len(sc.topicsToHubs[topic]))
	if set, exists := sc.topicsToHubs[topic]; exists {
		for url := range set {
			hubs = append(hubs, url)
		}
	}
	return hubs
}

// SubscribeToTopic pings the topic url associated with a topic.
// If that topic has no associated, returns an error
// Handles redirect responses (307 and 308) gracefully
// Passes any errors up, gracefully
func (sc *Client) SubscribeToTopic(topic string) error {
	// I'm not confident that this is how we want to get the URL
	if topicURL, ok := sc.topicsToSelf[topic]; ok {

		// Prepare the body
		data := make(url.Values)
		data.Set("hub.callback", sc.callback)
		data.Set("hub.mode", "subscribe")
		data.Set("hub.topic", topicURL)

		// Form the request
		req, _ := http.NewRequest("POST", topicURL, strings.NewReader(data.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.Header.Set("Content-Length", strconv.Itoa(len(data.Encode())))

		// Make the request
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			panic(err)
		}

		// Process the response, including any redirects or errors
		if respBody, err := ioutil.ReadAll(resp.Body); err == nil {
			switch resp.StatusCode {
			case 202:
				log.Printf("Successfully submitted subscription request to topic %s on url %s, pending validation", topic, topicURL)
				sc.pendingSubs[topicURL] = struct{}{}
				return nil
			case 307:
				log.Printf("Temporary redirect response, trying new address...")
				return sc.SubscribeToTopic(resp.Header.Get("Location"))
			case 308:
				log.Printf("Permanent redirect response, trying new address...")
				return sc.SubscribeToTopic(resp.Header.Get("Location"))
			default:
				return fmt.Errorf("Error in making subscription.  Code {%d}, Header{%v}, Details {%s}",
					resp.StatusCode, resp.Header, respBody)
			}
		} else {
			return err
		}
	}
	return errors.New("No URL known for the given topic")
}

// Callback is the function that is hit when a hub responds to
// a sub/un-sub request.
func (sc *Client) Callback(w http.ResponseWriter, req *http.Request) {
	// Differentiate between verification and denial notifications
	query := req.URL.Query()

	switch query.Get("hub.mode") {
	case "denied":
		topic := query.Get("hub.topic")
		reason := query.Get("hub.reason")
		log.Printf("Subscription to topic %s from url %s rejected.  Reason provided: {%s}", topic, req.Host, reason)
	case "subscribe":
		topic := query.Get("hub.topic")
		challenge := query.Get("hub.challenge")
		leaseSeconds := query.Get("hub.lease_seconds")
		log.Printf("Subscription to topic %s from url %s verification begin.  Challenge provided: {%s}.  Lease length (s): {%s}", topic, req.Host, challenge, leaseSeconds)

		// Turns out we have to store the fact that a subscription request was made
		if _, exists := sc.pendingSubs[topic]; exists {

			// Immediately spawn a renewal thread
			go func() {
				seconds, err := strconv.Atoi(leaseSeconds)
				if err != nil {
					panic(err)
				}
				// Sleep for some proportion of the lease time
				time.Sleep(time.Duration(2*seconds/3) * time.Second)

				// If there's an error, log and delete the topic from subscribed list
				if err = sc.SubscribeToTopic(topic); err != nil {
					log.Printf("Encountered an error while renewing subscription {%v}", err)
					delete(sc.activeSubs, topic)
				}

			}()

			// Write our response
			w.WriteHeader(200)
			w.Write([]byte(challenge))
			delete(sc.pendingSubs, topic)

			// Add to active subs, if we're not already active
			if _, allocated := sc.activeSubs[topic]; !allocated {
				sc.activeSubs[topic] = struct{}{}
			}

			return
		}

		// Received a callback for a function that we did not send
		w.WriteHeader(404)
		w.Write([]byte(""))

	case "unsubscribe":
		topic := query.Get("hub.topic")
		challenge := query.Get("hub.challenge")
		log.Printf("Unsubscribe from topic %s from url %s verification begin.  Challenge provided: {%s}.", topic, req.Host, challenge)

		if _, exists := sc.pendingUnSubs[topic]; exists {
			w.WriteHeader(200)
			w.Write([]byte(challenge))
			delete(sc.pendingSubs, topic)
			return
		}

		// Received a callback for a function that we did not send
		w.WriteHeader(404)
		w.Write([]byte(""))
	default:
		// This indicates a broken request.
	}
}

// DiscoverTopic runs the common discovery algorithm, and compiles its results into the client map
func (sc *Client) DiscoverTopic(topic string) {
	hubs, self := discovery.DiscoverTopic(topic)

	// Allocate the map if necessary
	if _, ok := sc.topicsToHubs[topic]; !ok {
		sc.topicsToHubs[topic] = make(map[string]struct{})
	}

	// Iterate through the results
	for hub := range hubs {
		sc.topicsToHubs[topic][hub] = struct{}{}
	}
	sc.topicsToSelf[topic] = self
}
