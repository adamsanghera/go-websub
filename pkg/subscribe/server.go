/*
package subscribe is a Go Client that implements the W3 Group's
WebSub protocol (https://www.w3.org/TR/websub/), a broker-supported pub-sub
architecture built on top of HTTP.

Check out more high-level information here: https://github.com/adamsanghera/go-websub/tree/master/cmd/subscriber
*/
package subscribe

import (
	"context"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/adamsanghera/go-websub/pkg/discovery"
)

// subscription represents an active subscription
type activeSubscription struct {
	callbackURL string
	cancel      context.CancelFunc
}

// Server creates, maintains, and release to topic hubs, following the websub protocol
type Server struct {
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

// NewServer creates and returns a new subscription Client
// Callback needs to be formatted like http{s}://website.domain:{port}/endpoint
func NewServer(port string) *Server {
	// Create the Client
	client := &Server{
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
			log.Fatalf("Callback Client crashed %v\n", err)
		}
	}()

	// Add some delay to give the Client time to spin up
	time.Sleep(1 * time.Millisecond)

	return client
}

// GetHubsForTopic returns all hubs associated with a given topic
func (sc *Server) GetHubsForTopic(topic string) []string {
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

// DiscoverTopic runs the common discovery algorithm, and compiles its results into the Client map
func (sc *Server) DiscoverTopic(topic string) error {
	hubs, self, err := discovery.DiscoverTopic(topic)

	if err != nil {
		return err
	}

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

	return nil
}

// Shutdown is called to indicate that a Client is no longer going to be used.
// It sends a shutdown signal to the Client's callback Client, freeing up the port to be used by another service.
func (sc *Server) Shutdown() {
	if err := sc.callbackSrv.Shutdown(context.Background()); err != nil {
		log.Fatalf("Failed to shutdown callback Client %v\n", err)
	}

	sc.aSubsMut.Lock()
	// Halt all of our active subscriptions
	for _, sub := range sc.activeSubs {
		sub.cancel()
	}
	sc.aSubsMut.Unlock()

	time.Sleep(50 * time.Millisecond)

	sc.aSubsMut.Lock()
	if len(sc.activeSubs) != 0 {
		panic("Failed to cancel all subs")
	}
	sc.aSubsMut.Unlock()
}
