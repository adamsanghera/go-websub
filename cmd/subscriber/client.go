/*
Package subscriber is a Go client library that implements the W3 Group's
WebSub protocol (https://www.w3.org/TR/websub/), a broker-supported pub-sub
architecture built on top of HTTP.

According to https://www.w3.org/TR/websub/#subscriber, a Subscriber
is a service that discovers hubs, and subscribes to topics.

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

This package implements the above requirements with the Client struct.

The client has three stages in its life cycle.

1. Birth
   - All data structures are initialized
   - An http server is created, to support callbacks (https://www.w3.org/TR/websub/#hub-verifies-intent)
   - The callback endpoint is registered
2. Normal state
   - Processes subscription/unsubscription/discovery commands in parallel
   - Should never panic, only log errors
3. Shutdown
   - Sends a shutdown signal to the client's callback server

Assumptions:
	- Cient is a long-running service
	- Sticky subscriptions (i.e. auto-renewing subscriptions) are the only subscriptions we want
*/
package subscriber

import (
	"context"
	"log"
	"net/http"
	"sync"

	"github.com/adamsanghera/go-websub/internal/discovery"
)

// Client subscribes to topic hubs, following the websub protocol
type Client struct {
	port         string
	topicsToHubs map[string]map[string]struct{}
	topicsToSelf map[string]string

	pendingSubs   map[string]string // perhaps point to a boolean sticky/not-sticky?
	pendingUnSubs map[string]struct{}
	activeSubs    map[string]struct{}

	tthMut *sync.Mutex
	ttsMut *sync.Mutex

	pSubsMut   *sync.Mutex
	pUnSubsMut *sync.Mutex
	aSubsMut   *sync.Mutex

	callbackMux *http.ServeMux
	callbackSrv *http.Server
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

		tthMut: &sync.Mutex{},
		ttsMut: &sync.Mutex{},

		pSubsMut:   &sync.Mutex{},
		pUnSubsMut: &sync.Mutex{},
		aSubsMut:   &sync.Mutex{},

		callbackMux: http.NewServeMux(),
		callbackSrv: &http.Server{Addr: ":" + port},
	}

	client.callbackSrv.Handler = client.callbackMux

	go func() {
		client.callbackMux.HandleFunc("/callback/", client.CallbackSwitch)
		// Handles all callbacks for subscriptions, unsubscriptions, etc.
		if err := client.callbackSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Callback server crashed %v\n", err)
		}
	}()

	return client
}

// GetHubsForTopic returns all hubs associated with a given topic
func (sc *Client) GetHubsForTopic(topic string) []string {
	sc.tthMut.Lock()
	defer sc.tthMut.Unlock()

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

	sc.tthMut.Lock()
	defer sc.tthMut.Unlock()
	sc.ttsMut.Lock()
	defer sc.ttsMut.Unlock()

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

// Shutdown is called to indicate that a Client is no longer going to be used.
// It sends a shutdown signal to the Client's callback server, freeing up the port to be used by another service.
func (sc *Client) Shutdown() {
	if err := sc.callbackSrv.Shutdown(context.Background()); err != nil {
		log.Fatalf("Failed to shutdown callback server %v\n", err)
	}
}
