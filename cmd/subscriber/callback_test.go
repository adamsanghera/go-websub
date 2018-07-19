package subscriber

import (
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"testing"

	httpmock "gopkg.in/jarcoal/httpmock.v1"
)

func TestClient_handleAckedSubscription(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	var callback string

	httpmock.RegisterResponder("POST", "http://example.com/feed",
		func(req *http.Request) (*http.Response, error) {
			resp := httpmock.NewStringResponse(202, "")
			if respBody, err := ioutil.ReadAll(req.Body); err == nil {
				log.Println(string(respBody))
			}
			return resp, nil
		})

	sc := NewClient("4000")
	t.Run("Everything works", func(t *testing.T) {
		sc.topicsToSelf["http://example.com/feed"] = "http://example.com/feed"
		sc.SubscribeToTopic("http://example.com/feed")

		if _, ok := sc.pendingSubs["http://example.com/feed"]; !ok {
			t.Fatal("Subscription not registered as pending")
		}

		// time.Sleep(1 * time.Second)

		if len(callback) == 0 {
			t.Fatal("Callback unset")
		}

		// At this point, the callback URI is up and waiting
		data := make(url.Values)
		data.Set("hub.mode", "subscribe")
		data.Set("hub.topic", "http://example.com/feed")
		data.Set("hub.challenge", "kitties")
		data.Set("hub.lease_seconds", "20")

		req, _ := http.NewRequest("POST", "localhost:4000/"+callback, strings.NewReader(data.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.Header.Set("Content-Length", strconv.Itoa(len(data.Encode())))

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			panic(err)
		}

		if resp.StatusCode != 200 {
			t.Fatalf("Status code is %d instead of 200", resp.StatusCode)
		}

		if respBody, err := ioutil.ReadAll(resp.Body); err == nil {
			if string(respBody) != "kitties" {
				t.Fatalf("Response is {%v} instead of {kitties}", respBody)
			}
		} else {
			t.Fatalf("Failed to parse body with err {%v}", err)
		}

	})
}
