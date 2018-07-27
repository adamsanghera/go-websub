package subscriber

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

// Subscribe pings the topic url associated with a topic.
// If that topic has no associated, returns an error
// Handles redirect responses (307 and 308) gracefully
// Passes any errors up, gracefully
func (sc *Client) Subscribe(topic string) error {

	sc.ttsMut.Lock()

	// NOTE(adam): I'm not confident that this is how we want to get or store these urls
	if topicURL, ok := sc.topicsToSelf[topic]; ok {

		sc.ttsMut.Unlock() // It is ok if this state changes from underneath us

		callback := generateCallback()
		req := buildSubscriptionRequest(callback, topicURL)

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			panic(err)
		}

		return sc.processSubscriptionResponse(resp, topicURL, callback)
	}

	sc.ttsMut.Unlock()
	return errors.New("No URL known for the given topic")
}

// processes a response to the subscription request
func (sc *Client) processSubscriptionResponse(resp *http.Response, topicURL string, callbackURI string) error {
	if respBody, err := ioutil.ReadAll(resp.Body); err == nil {
		switch resp.StatusCode {
		case 202:
			sc.pSubsMut.Lock()
			sc.pendingSubs[topicURL] = callbackURI
			sc.pSubsMut.Unlock()
			return nil
		case 307:
			log.Printf("Temporary redirect response, trying new address...")
			return sc.Subscribe(resp.Header.Get("Location"))
		case 308:
			log.Printf("Permanent redirect response, trying new address...")
			return sc.Subscribe(resp.Header.Get("Location"))
		default:
			return fmt.Errorf("Error in making subscription.  Code {%d}, Header{%v}, Details {%s}",
				resp.StatusCode, resp.Header, respBody)
		}
	} else {
		return err
	}
}

// helper function to generate a 16-byte (32 chars) string
func generateCallback() string {
	randomURI := make([]byte, 16)
	rand.Read(randomURI)
	return hex.EncodeToString(randomURI)
}

// builds a pub-sub compliant subscription request, given a topic url and callback
func buildSubscriptionRequest(callback string, topic string) *http.Request {
	data := make(url.Values)
	data.Set("hub.callback", callback)
	data.Set("hub.mode", "subscribe")
	data.Set("hub.topic", topic)

	req, _ := http.NewRequest("POST", topic, strings.NewReader(data.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Content-Length", strconv.Itoa(len(data.Encode())))

	return req
}
