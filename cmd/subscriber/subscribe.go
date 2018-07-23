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

// SubscribeToTopic pings the topic url associated with a topic.
// If that topic has no associated, returns an error
// Handles redirect responses (307 and 308) gracefully
// Passes any errors up, gracefully
func (sc *Client) SubscribeToTopic(topic string) error {

	sc.ttsMut.Lock()

	// I'm not confident that this is how we want to get the URL
	if topicURL, ok := sc.topicsToSelf[topic]; ok {

		sc.ttsMut.Unlock() // It is ok if this state changes from underneath us

		// Generate some random data
		data := make(url.Values)
		randomURI := make([]byte, 16)
		rand.Read(randomURI)

		// Prepare the body
		data.Set("hub.callback", hex.EncodeToString(randomURI))
		data.Set("hub.mode", "subscribe")
		data.Set("hub.topic", topicURL)

		// Form the request
		req, _ := http.NewRequest("POST", topicURL, strings.NewReader(data.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.Header.Set("Content-Length", strconv.Itoa(len(data.Encode())))

		// Make the request
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			panic(err)
		}

		return sc.processSubscriptionResponse(resp, topicURL, hex.EncodeToString(randomURI), topic)
	}

	sc.ttsMut.Unlock()
	return errors.New("No URL known for the given topic")
}

func (sc *Client) processSubscriptionResponse(
	resp *http.Response, topicURL string, callbackURI string, topic string) error {
	// Process the response, including any redirects or errors
	if respBody, err := ioutil.ReadAll(resp.Body); err == nil {
		switch resp.StatusCode {
		case 202:
			log.Printf("Successfully submitted subscription request to topic %s on url %s, pending validation", topic, topicURL)
			sc.pSubsMut.Lock()
			defer sc.pSubsMut.Unlock()
			sc.pendingSubs[topicURL] = callbackURI
			return nil
		case 307:
			log.Printf("Temporary redirect response, trying new address...")
			return sc.SubscribeToTopic(resp.Header.Get("Location"))
		case 308:
			log.Printf("Permanent redirect response, trying new address...")
			return sc.SubscribeToTopic(resp.Header.Get("Location"))
		default:
			return fmt.Errorf("Error in making subscription.  Code {%d}, Header{%v}, Details {%s}",
				resp.StatusCode, resp.Header, respBody)
		}
	} else {
		return err
	}
}
