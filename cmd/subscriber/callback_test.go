package subscriber

import (
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"testing"
	"time"

	httpmock "gopkg.in/jarcoal/httpmock.v1"
)

var topicURL = "http://example.com/feed"

func TestSuccessfulSubscription(t *testing.T) {
	httpmock.Activate()
	var callback string
	sc := NewClient("4000")

	// When the client attempts to subscribe to the topicURL, they'll get a 202
	RegisterSuccessfulValidationAck(t, &callback)

	t.Run("Everything works", func(t *testing.T) {
		sc.topicsToSelf[topicURL] = topicURL

		// The POST is made in here
		sc.Subscribe(topicURL)

		sc.pSubsMut.Lock()
		if _, ok := sc.pendingSubs[topicURL]; !ok {
			t.Fatal("Subscription not registered as pending")
		}
		sc.pSubsMut.Unlock()

		VerifyCallback(t, topicURL, callback)

		sc.aSubsMut.Lock()
		if _, exists := sc.activeSubs[topicURL]; !exists {
			t.Fatal("Subscription is not in active set, even though it was accepted")
		}
		sc.aSubsMut.Unlock()

		sc.pSubsMut.Lock()
		if _, exists := sc.pendingSubs[topicURL]; exists {
			t.Fatal("Subscription is in pending set, even though it was accepted")
		}
		sc.pSubsMut.Unlock()
	})

	sc.Shutdown()
	httpmock.DeactivateAndReset()
}

// TODO(adam): Testing parallel subscription handling
func TestImmediatelyDeniedSubscription(t *testing.T) {
	httpmock.Activate()
	var callback string
	sc := NewClient("4000")

	// When the client attempts to subscribe to the topicURL, they'll get a 202
	RegisterSuccessfulValidationAck(t, &callback)

	t.Run("Subscription immediately denied", func(t *testing.T) {
		sc.topicsToSelf[topicURL] = topicURL

		// The POST is made in here
		sc.Subscribe(topicURL)

		if _, ok := sc.pendingSubs[topicURL]; !ok {
			t.Fatal("Subscription not registered as pending")
		}

		DenyCallback(t, topicURL, callback)

		sc.pSubsMut.Lock()
		if _, exists := sc.pendingSubs[topicURL]; exists {
			t.Fatalf("Subscription in pending, even though it was denied")
		}
		sc.pSubsMut.Unlock()

		sc.aSubsMut.Lock()
		if _, exists := sc.activeSubs[topicURL]; exists {
			t.Fatal("Subscription is in active set, even though it was denied")
		}
		sc.aSubsMut.Unlock()
	})

	sc.Shutdown()
	httpmock.DeactivateAndReset()
}

func TestStaticUnsubscribe(t *testing.T) {
	// TODO(adam): Test unsubscribe
}

func TestDenialOnActiveSubscription(t *testing.T) {
	// TODO(adam): Test upstream denial of existing subscription.
	httpmock.Activate()
	var callback string
	sc := NewClient("4000")

	RegisterSuccessfulValidationAck(t, &callback)

	t.Run("1", func(t *testing.T) {
		sc.ttsMut.Lock()
		sc.topicsToSelf[topicURL] = topicURL
		sc.ttsMut.Unlock()

		// The POST request is made in here
		sc.Subscribe(topicURL)

		sc.pSubsMut.Lock()
		if _, ok := sc.pendingSubs[topicURL]; !ok {
			t.Fatal("Subscription not registered as pending")
		}
		sc.pSubsMut.Unlock()

		VerifyCallback(t, topicURL, callback)

		sc.pSubsMut.Lock()
		if _, exists := sc.pendingSubs[topicURL]; exists {
			t.Fatalf("Subscription is in pending set, even though it was accepted")
		}
		sc.pSubsMut.Unlock()

		sc.aSubsMut.Lock()
		if _, exists := sc.activeSubs[topicURL]; !exists {
			t.Fatal("Subscription is not in active set, even though it was accepted")
		}
		sc.aSubsMut.Unlock()

		DenyCallback(t, topicURL, callback)

		time.Sleep(3 * time.Second)

		sc.aSubsMut.Lock()
		if _, exists := sc.activeSubs[topicURL]; exists {
			t.Fatal("Subscription is in active set, even though it was denied")
		}
		sc.aSubsMut.Unlock()

		sc.pSubsMut.Lock()
		if _, exists := sc.pendingSubs[topicURL]; exists {
			t.Fatal("Subscription is in pending set, even though it was denied")
		}
		sc.pSubsMut.Unlock()

	})

	sc.Shutdown()
	httpmock.DeactivateAndReset()
	// Should cancel recurring re-sub requests
	// Should be removed from active/pending
}

// VerifyCallback will send a verification POST to the given callback.
// It expects to receive a parrotted challenge in response.
// httpmock is deactivated initially (to hit the callback), and reactivated on exit.
func VerifyCallback(t *testing.T, topicURL string, callback string) {
	// Turn off httpmock, so that we hit a live address
	httpmock.Deactivate()
	defer httpmock.Activate()

	// Message body
	data := make(url.Values)
	data.Set("hub.mode", "subscribe")
	data.Set("hub.topic", topicURL)
	data.Set("hub.challenge", "kitties")
	data.Set("hub.lease_seconds", "3")

	// Request itself
	req, err := http.NewRequest("POST", "http://localhost:4000/callback/"+callback, strings.NewReader(data.Encode()))
	if err != nil {
		panic(err)
	}

	// Headers
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Content-Length", strconv.Itoa(len(data.Encode())))

	// Make the request
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		panic(err)
	}

	// Should be a 200
	if resp.StatusCode != 200 {
		t.Fatalf("Status code is %d instead of 200", resp.StatusCode)
	}

	// Should have gotten the challenge parrotted back
	if respBody, err := ioutil.ReadAll(resp.Body); err == nil {
		if string(respBody) != "kitties" {
			t.Fatalf("Response is {%v} instead of {kitties}", respBody)
		}
	} else {
		t.Fatalf("Failed to parse body with err {%v}", err)
	}
}

// DenyCallback will send a rejection POST to the given callback
// It expects to receive an empty body back, with a 200 header.
// httpmock is deactivated initially (to hit the callback), and reactivated on exit.
func DenyCallback(t *testing.T, topicURL string, callback string) {
	// Turn off httpmock, so that we hit a live address
	httpmock.Deactivate()
	defer httpmock.Activate()

	// Message body
	data := make(url.Values)
	data.Set("hub.mode", "denied")
	data.Set("hub.topic", topicURL)

	// Request itself
	req, err := http.NewRequest("POST", "http://localhost:4000/callback/"+callback, strings.NewReader(data.Encode()))
	if err != nil {
		panic(err)
	}

	// Headers
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Content-Length", strconv.Itoa(len(data.Encode())))

	// Make the request
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		panic(err)
	}

	// Should be a 200
	if resp.StatusCode != 200 {
		t.Fatalf("Status code is %d instead of 200", resp.StatusCode)
	}

	// Should have gotten the empty string
	if respBody, err := ioutil.ReadAll(resp.Body); err == nil {
		if string(respBody) != "" {
			t.Fatalf("Response is {%v} instead of empty string", respBody)
		}
	} else {
		t.Fatalf("Failed to parse body with err {%v}", err)
	}
}

func RegisterSuccessfulValidationAck(t *testing.T, callback *string) {
	// When the subscription request is made, respond with a 202
	httpmock.RegisterResponder("POST", topicURL,
		func(req *http.Request) (*http.Response, error) {

			// Respond with a happy body.
			resp := httpmock.NewStringResponse(202, "")

			// Read the callback url into our test-global variable, callback
			if reqBody, err := ioutil.ReadAll(req.Body); err == nil {
				if values, err := url.ParseQuery(string(reqBody)); err == nil {
					*callback = values.Get("hub.callback")
				} else {
					panic(err)
				}
			}

			return resp, nil
		})

	// Give the super fast Travis servers time to let our client register its callback handler
	time.Sleep(5 * time.Millisecond)
}
