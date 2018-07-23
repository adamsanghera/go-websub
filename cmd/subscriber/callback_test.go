package subscriber

import (
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"testing"

	httpmock "gopkg.in/jarcoal/httpmock.v1"
)

func TestClient_handleAckedSubscription(t *testing.T) {
	httpmock.Activate()
	var callback string

	sc := NewClient("4000")

	// POSTs to this address will result in the
	httpmock.RegisterResponder("POST", "http://example.com/feed",
		func(req *http.Request) (*http.Response, error) {

			// Respond with a happy body.
			resp := httpmock.NewStringResponse(202, "")

			// Read the callback url into our test-global variable, callback
			if reqBody, err := ioutil.ReadAll(req.Body); err == nil {
				if values, err := url.ParseQuery(string(reqBody)); err == nil {
					callback = values.Get("hub.callback")
				} else {
					panic(err)
				}
			}

			// ACK the POST, now that we have the callback url...
			return resp, nil
		})

	t.Run("Everything works", func(t *testing.T) {
		sc.topicsToSelf["http://example.com/feed"] = "http://example.com/feed"

		// The POST is made in here
		sc.SubscribeToTopic("http://example.com/feed")

		if _, ok := sc.pendingSubs["http://example.com/feed"]; !ok {
			t.Fatal("Subscription not registered as pending")
		}

		if len(callback) == 0 {
			t.Fatal("Callback unset")
		}

		// Generate a message to send to the callback url

		// Message body
		data := make(url.Values)
		data.Set("hub.mode", "subscribe")
		data.Set("hub.topic", "http://example.com/feed")
		data.Set("hub.challenge", "kitties")
		data.Set("hub.lease_seconds", "20")

		// Request itself
		req, err := http.NewRequest("POST", "http://localhost:4000/callback/"+callback, strings.NewReader(data.Encode()))
		if err != nil {
			panic(err)
		}

		// Headers
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.Header.Set("Content-Length", strconv.Itoa(len(data.Encode())))

		// Turn off httpmock, so that we hit a live address
		httpmock.Deactivate()

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
	})

	sc.ShutDown()
	httpmock.DeactivateAndReset()
}

// TODO(adam): Testing parallel subscription handling
