/*
Package subscribe is a Go Server that implements the W3 Group's
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

// NewServer creates and returns a new subscription Server
// Callback needs to be formatted like http{s}://website.domain:{port}/endpoint
func NewServer(port string) *Server {
	// Create the Server
	srv := &Server{
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

	srv.callbackSrv.Handler = srv.callbackMux

	// Handles all callbacks for subscriptions, unsubscriptions, etc.
	srv.callbackMux.HandleFunc("/callback/", srv.CallbackSwitch)

	go func() {
		if err := srv.callbackSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Callback srv crashed %v\n", err)
		}
	}()

	// Add some delay to give the srv time to spin up
	time.Sleep(1 * time.Millisecond)

	return srv
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

// Shutdown is called to indicate that a Server is no longer going to be used.
// It sends a shutdown signal to the Server's callback Server, freeing up the port to be used by another service.
func (sc *Server) Shutdown() {
	if err := sc.callbackSrv.Shutdown(context.Background()); err != nil {
		log.Fatalf("Failed to shutdown callback Server %v\n", err)
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
