package subscriber

import (
	"bufio"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/adamsanghera/go-websub/common"
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
}

// NewClient creates and returns a new subscription client
// Callback needs to be formatted like http{s}://website.domain:{port}/endpoint
func NewClient(callback string) *Client {
	return &Client{
		callback:     callback,
		topicsToHubs: make(map[string]map[string]struct{}),
		topicsToSelf: make(map[string]string),
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

// SubscribeToTopic pings the
func (sc *Client) SubscribeToTopic(topic string) {
	for _, postURL := range sc.topicsToSelf {
		data := make(url.Values)
		data.Set("hub.callback", sc.callback)
		data.Set("hub.mode", "subscribe")
		data.Set("hub.topic", postURL)

		log.Printf("Pinging %v\n", postURL)

		req, _ := http.NewRequest("POST", postURL, strings.NewReader(data.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.Header.Set("Content-Length", strconv.Itoa(len(data.Encode())))

		// Make the request
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			panic(err)
		}

		s, _ := bufio.NewReader(resp.Body).ReadString(byte('\t'))

		log.Printf("Response received with resp {%s}, header {%v}, code {%v}",
			s, resp.Header, resp.StatusCode)
	}
}

// DiscoverTopic runs the common discovery algorithm, and compiles its results into the client map
func (sc *Client) DiscoverTopic(topic string) {
	hubs, self := common.DiscoverTopic(topic)

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
