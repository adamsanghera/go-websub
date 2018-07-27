/*
Package subscriber is a Go client library that implements the W3 Group's
WebSub protocol (https://www.w3.org/TR/websub/), a broker-supported pub-sub
architecture built on top of HTTP.

Check out more high-level information here: https://github.com/adamsanghera/go-websub/tree/master/cmd/subscriber
*/
package subscriber

import (
	"context"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/adamsanghera/go-websub/internal/discovery"
)

// subscription represents an active subscription
type activeSubscription struct {
	callbackURL string
	cancel      context.CancelFunc
}

// Client creates, maintains, and release to topic hubs, following the websub protocol
type Client struct {
	port string

	// vars related to discovery
	topicsToHubs map[string]map[string]struct{}
	topicsToSelf map[string]string
	tthMut       *sync.Mutex
	ttsMut       *sync.Mutex

	// vars related to (un)subscriptions
	pendingSubs   map[string]string // perhaps point to a boolean sticky/not-sticky?
	pendingUnSubs map[string]string
	activeSubs    map[string]*activeSubscription // Holds the cancel funcs for sticky subs
	pSubsMut      *sync.Mutex
	pUnSubsMut    *sync.Mutex
	aSubsMut      *sync.Mutex

	callbackMux *http.ServeMux
	callbackSrv *http.Server
	// TODO(adam) manage secrets per topic
}

// NewClient creates and returns a new subscription client
// Callback needs to be formatted like http{s}://website.domain:{port}/endpoint
func NewClient(port string) *Client {
	// Create the client
	client := &Client{
		port: port,

		topicsToHubs: make(map[string]map[string]struct{}),
		topicsToSelf: make(map[string]string),
		tthMut:       &sync.Mutex{},
		ttsMut:       &sync.Mutex{},

		pendingSubs:   make(map[string]string),
		pendingUnSubs: make(map[string]string),
		activeSubs:    make(map[string]*activeSubscription),
		pSubsMut:      &sync.Mutex{},
		pUnSubsMut:    &sync.Mutex{},
		aSubsMut:      &sync.Mutex{},

		callbackMux: http.NewServeMux(),
		callbackSrv: &http.Server{Addr: ":" + port},
	}

	client.callbackSrv.Handler = client.callbackMux

	client.callbackMux.HandleFunc("/callback/", client.CallbackSwitch)
	// Handles all callbacks for subscriptions, unsubscriptions, etc.

	go func() {
		if err := client.callbackSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Callback server crashed %v\n", err)
		}
	}()

	// Add some delay to give the server time to spin up
	time.Sleep(1 * time.Millisecond)

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
