package subscriber

import (
	"context"
	"database/sql"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"testing"

	"gopkg.in/jarcoal/httpmock.v1"
)

func TestSubscriber_subscribe_redirect(t *testing.T) {
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

	redirectDest := "http://temp_hub.com/hub"

	setupTempRedirect(hubURLTest, redirectDest, t, func(req *http.Request) error {
		reqBody, err := ioutil.ReadAll(req.Body)
		if err != nil {
			return err
		}
		data, err := url.ParseQuery(string(reqBody))
		if err != nil {
			return err
		}
		if data.Get("hub.mode") != "subscribe" {
			return fmt.Errorf("Bad mode %v instead of %v", data.Get("hub.mode"), "subscribe")
		}
		if data.Get("hub.topic") != topicURLTest {
			return fmt.Errorf("Bad topic %v instead of %v", data.Get("hub.topic"), topicURLTest)
		}
		return nil
	})

	setupDummyValidationAck(redirectDest)

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

	err = sub.Shutdown()
	if err != nil {
		t.Fatal(err)
	}

	httpmock.DeactivateAndReset()
}

func TestSubscriber_subscribe_cancelledCtx(t *testing.T) {
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

	setupDummyValidationAck(hubURLTest)

	err = sub.storage.IndexOffer(map[string]string{
		topicURLTest: hubURLTest,
	})
	if err != nil {
		t.Fatal(err)
	}

	// This can happen when
	// (1) user cancels after subscribing
	// (2) cancellation occurs before renewal
	badCtx, cancel := context.WithCancel(context.Background())
	cancel()

	// We would want subscribe to return cancelled context, and for no subscription to be created.
	err = sub.initiateSubscription(badCtx, topicURLTest, hubURLTest)
	if err != context.Canceled {
		t.Fatal(err)
	}

	_, err = sub.storage.GetActiveCallback(topicURLTest, hubURLTest)
	if err != sql.ErrNoRows {
		t.Fatalf("Expected {%v} but got {%v}", sql.ErrNoRows, err)
	}

	err = sub.Shutdown()
	if err != nil {
		t.Fatal(err)
	}

	httpmock.DeactivateAndReset()
}
