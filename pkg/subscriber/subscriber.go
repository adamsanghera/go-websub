/*
Package subscriber is a Go Server that implements the W3 Group's
WebSub protocol (https://www.w3.org/TR/websub/), a broker-supported pub-sub
architecture built on top of HTTP.

Check out more high-level information here: https://github.com/adamsanghera/go-websub/tree/master/cmd/subscriber
*/
package subscriber

import (
	"context"
	"fmt"
	"net/http"

	"github.com/adamsanghera/go-websub/pkg/subscriber/api"

	"github.com/adamsanghera/go-websub/pkg/subscriber/storage/sql"
)

// Subscriber creates, maintains, and release to topic hubs, following the websub protocol
type Subscriber struct {
	// Client, to make calls to hubs
	transport *http.Transport
	client    *http.Client

	// Server and mux, to handle callbacks
	callbackMux *http.ServeMux
	callbackSrv *http.Server

	websub api.API

	// Centralized source of truth for subscriptions
	storage *sql.SQL

	// Sticky subscription manager
	stickySubscriptions map[string]context.CancelFunc
}

// New creates and returns a new Subscriber from a given config object
func New(cfg *Config) (*Subscriber, error) {
	// Init our storage system
	storage, err := sql.New(sql.NewConfig())
	if err != nil {
		return nil, err
	}

	// Init the http server needed to support callbacks
	callbackMux := http.NewServeMux()
	callbackSrv := &http.Server{Addr: ":" + cfg.port}
	callbackSrv.Handler = callbackMux

	// Init transport and client
	transport := &http.Transport{}
	client := &http.Client{
		Transport: transport,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	return &Subscriber{
		transport:           transport,
		client:              client,
		callbackMux:         callbackMux,
		callbackSrv:         callbackSrv,
		storage:             storage,
		stickySubscriptions: make(map[string]context.CancelFunc),
	}, nil
}

// Run starts the Subscriber's Callback Server,
//   which effectively means that the subscriber is on.
func (sub *Subscriber) Run() error {
	sub.callbackMux.HandleFunc("/callback/", sub.callbackSwitch)
	return sub.callbackSrv.ListenAndServe()
}

// GetHubsForTopic returns all hubs associated with a given topic
// func (sc *Server) GetHubsForTopic(topic string) []string {
// 	sc.tthMut.Lock()
// 	defer sc.tthMut.Unlock()

// 	hubs := make([]string, len(sc.topicsToHubs[topic]))
// 	if set, exists := sc.topicsToHubs[topic]; exists {
// 		for url := range set {
// 			hubs = append(hubs, url)
// 		}
// 	}
// 	return hubs
// }

// Shutdown is called to indicate that a Server is no longer going to be used.
// It sends a shutdown signal to the Server's callback Server, freeing up the port to be used by another service.
func (sub *Subscriber) Shutdown() error {
	if err := sub.callbackSrv.Shutdown(context.Background()); err != nil {
		return fmt.Errorf("Failed to shutdown callback Server %v", err)
	}

	return sub.storage.Shutdown()
	// TODO(adam) cancel all of our subscriptions
}
