package subscriber

import (
	"context"
	"database/sql"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	httpmock "gopkg.in/jarcoal/httpmock.v1"
)

var topicURLTest = "http://example.com/topic"
var hubURLTest = "http://example.com/hub"

func TestSuccessfulSubscription(t *testing.T) {
	sub, err := New(NewConfig())
	if err != nil {
		t.Fatal(err)
	}
	httpmock.ActivateNonDefault(sub.client)

	go func(t *testing.T) {
		if err := sub.Run(); err != http.ErrServerClosed {
			t.Fatal(err)
		}
	}(t)

	var callback string

	httpmock.RegisterResponder("POST", hubURLTest,
		func(req *http.Request) (*http.Response, error) {
			bdy, _ := ioutil.ReadAll(req.Body)
			vals, _ := url.ParseQuery(string(bdy))
			callback = vals.Get("hub.callback")
			return &http.Response{
				StatusCode: 202,
			}, nil
		})

	err = sub.storage.IndexOffer(map[string]string{
		topicURLTest: hubURLTest,
	})
	if err != nil {
		t.Fatal(err)
	}

	err = sub.initiateSubscription(context.Background(), topicURLTest, hubURLTest)
	if err != nil {
		t.Fatal(err)
	}

	if callback == "" {
		t.Fatalf("Expected callback to not be empty")
	}

	data := make(url.Values)
	data.Set("hub.mode", "subscribe")
	data.Set("hub.lease_seconds", "2")
	data.Set("hub.challenge", "121412")

	req, _ := http.NewRequest("POST", "http://localhost:4000/callback/"+callback, strings.NewReader(data.Encode()))

	testClient := http.Client{}
	resp, err := testClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}

	if resp.StatusCode != 200 {
		t.Fatalf("Expected code 200 but received %d", resp.StatusCode)
	}
	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	if string(respBody) != data.Get("hub.challenge") {
		t.Fatalf("Expected response body {%v} but received {%v}", data.Get("hub.challenge"), string(respBody))
	}

	cb, err := sub.storage.GetActiveCallback(topicURLTest, hubURLTest)
	if err != nil {
		t.Fatal(err)
	}
	if cb != callback {
		t.Fatalf("Expected stored cb %v to be equal to sent cb %v", cb, callback)
	}

	// Cancel the subscription, wait for the lease to expire, check to see that it's no longer active
	sub.stickySubscriptions[cb]()
	time.Sleep(3 * time.Second)
	_, err = sub.storage.GetActiveCallback(topicURLTest, hubURLTest)
	if err != sql.ErrNoRows {
		t.Fatal(err)
	}
}
