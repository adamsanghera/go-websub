package http

import "net/http"

// WebSub is a wrapper of an http server and client, which together implement websub
type WebSub struct {
	srv    *http.Server // Used to listen for callbacks
	client *http.Client // Used to send subscription requests to other http servers
}

// Discover searches for a given topic url
func (ws *WebSub) Discover(topic string) error {

}

// Subscribe does what it says
func (ws *WebSub) Subscribe(topic, hub string) error {

}

// ReceiveCallback does what it says
func (ws *WebSub) ReceiveCallback() <-chan string {

}
