package subscribe

import (
	"errors"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

// unsubscribe unsubs from the given topic url.
// If that topic has no associated url, returns an error
// Handles redirect responses (307 and 308) gracefully
// Gracefully passes any errors up
func (sc *Server) unsubscribe(topic string) error {
	sc.ttsMut.Lock()

	// NOTE(adam): I'm not confident that this is how we want to get or store these urls
	if topicURL, ok := sc.topicsToSelf[topic]; ok {

		sc.ttsMut.Unlock() // It is ok if this state changes from underneath us

		callback := generateCallback()
		req := buildUnsubscriptionRequest(callback, topicURL)

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			panic(err)
		}

		return sc.processSubscriptionResponse(resp, topicURL, callback, true)
	}

	sc.ttsMut.Unlock()
	return errors.New("No URL known for the given topic")
}

// builds a pub-sub compliant subscription request, given a topic url and callback
func buildUnsubscriptionRequest(callback string, topic string) *http.Request {
	data := make(url.Values)
	data.Set("hub.callback", callback)
	data.Set("hub.mode", "unsubscribe")
	data.Set("hub.topic", topic)

	req, _ := http.NewRequest("POST", topic, strings.NewReader(data.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Content-Length", strconv.Itoa(len(data.Encode())))

	return req
}
