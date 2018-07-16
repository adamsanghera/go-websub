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
	callback     string
	topicsToHubs map[string]map[string]struct{}
	topicsToSelf map[string]string

	subscribedTopicURLs map[string]map[string]struct{}
}

// NewClient creates and returns a new subscription client
// Callback needs to be formatted like http{s}://website.domain:{port}/endpoint
func NewClient(callback string) *Client {
	return &Client{
		callback:     callback,
		topicsToHubs: make(map[string]map[string]struct{}),
		topicsToSelf: make(map[string]string),

		subscribedTopicURLs: make(map[string]map[string]struct{}),
	}
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
				log.Printf("Successfully subscribed to topic %s on url %s, pending validation", topic, topicURL)
				if _, ok := sc.subscribedTopicURLs[topic]; !ok {
					sc.subscribedTopicURLs[topic] = make(map[string]struct{})
				}
				sc.subscribedTopicURLs[topic][topicURL] = struct{}{}
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
