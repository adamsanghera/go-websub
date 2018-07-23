package subscriber

import (
	"context"
	"log"
	"net/http"

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
	topicsToHubs map[string]map[string]struct{}
	topicsToSelf map[string]string

	pendingSubs   map[string]string // perhaps point to a boolean sticky/not-sticky?
	pendingUnSubs map[string]struct{}
	activeSubs    map[string]struct{}

	srvMux *http.ServeMux
	srv    *http.Server
	// TODO(adam) manage secrets per topic
}

// NewClient creates and returns a new subscription client
// Callback needs to be formatted like http{s}://website.domain:{port}/endpoint
func NewClient(port string) *Client {
	// Create the client
	client := &Client{
		topicsToHubs: make(map[string]map[string]struct{}),
		topicsToSelf: make(map[string]string),

		pendingSubs:   make(map[string]string),
		pendingUnSubs: make(map[string]struct{}),
		activeSubs:    make(map[string]struct{}),
		srvMux:        http.NewServeMux(),
		srv:           &http.Server{Addr: ":4000"},
	}

	client.srv.Handler = client.srvMux

	go func() {
		client.srvMux.HandleFunc("/callback/", client.CallbackSwitch)
		// Handles all callbacks for subscriptions, unsubscriptions, etc.
		if err := client.srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Callback server crashed %v\n", err)
		}
	}()

	return client
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

func (sc *Client) ShutDown() {
	if err := sc.srv.Shutdown(context.Background()); err != nil {
		log.Fatalf("Failed to shutdown callback server %v\n", err)
	}
}
