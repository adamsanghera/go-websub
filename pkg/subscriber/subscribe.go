package subscriber

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

func (sub *Subscriber) initiateSubscription(ctx context.Context, topic, hub string) error {
	callback := generateCallback()

	resp, err := sub.sendSubscriptionRequest(topic, hub, callback)
	if err != nil {
		return err
	}

	code := resp.StatusCode

	// ACK
	if code == 202 {
		return sub.storage.NewCallback(ctx, topic, hub, callback)
	}

	// Redirect
	if code == 307 || code == 308 {
		newHubLoc := resp.Header.Get("Location")

		sub.storage.IndexOffer(map[string]string{topic: newHubLoc})
		if code == 307 {
			log.Printf("Temporary redirect response, to new address {%v}", newHubLoc)
		} else {
			// TODO(adam): Consider replacing old hub url in storage, instead of supplanting
			log.Printf("Permanent redirect response, to new address {%v}", newHubLoc)
		}
		return sub.initiateSubscription(ctx, topic, newHubLoc)
	}

	return fmt.Errorf("Invalid status code while making subscription request, resp: {%+v}", resp)
}

func (sub *Subscriber) sendSubscriptionRequest(topic, hub, callback string) (*http.Response, error) {
	data := make(url.Values)
	data.Set("hub.callback", callback)
	data.Set("hub.mode", "subscribe")
	data.Set("hub.topic", topic)

	req, _ := http.NewRequest("POST", hub, strings.NewReader(data.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Content-Length", strconv.Itoa(len(data.Encode())))

	resp, err := sub.client.Do(req)
	if err != nil {
		return nil, err
	}

	return resp, err
}

// renewSubscription is very similar to initiateSubscription.
// The difference is that renewSubscription explicitly re-uses the given callback.
// If the subscription lease associated with the given callback is found to be expired,
//  then this function will exit earl.
// If the lease expires AFTER this subscription request has been submitted, two things are possible:
//   1. The hub erroneously extends the lease.  In this case, the subscriber client will rejec the extension.
//      The client will send a 404 back to the hub, which should cause the two to agree that no valid lease exists.
//   2. The hub denies the lease.  This invalidates the existing lease, which is already expired from the client's pov.
//      The invalidation invocation is idempotent, so nothing bad happens.  The timestamp recorded as the final expiration
//      is pegged to the moment that the denial message was received.
func (sub *Subscriber) renewSubscription(ctx context.Context, callback string) error {
	// check to see that the subscription is still active.
	// this will return an error if it isn't
	subscription, err := sub.storage.GetSubscription(callback)
	if err != nil {
		return err
	}

	topic, hub := subscription.Topic, subscription.Hub

	resp, err := sub.sendSubscriptionRequest(topic, hub, callback)
	if err != nil {
		return err
	}

	code := resp.StatusCode

	// ACK
	if code == 202 {
		return nil
	}

	// Redirect
	if code == 307 || code == 308 {
		newHubLoc := resp.Header.Get("Location")

		sub.storage.IndexOffer(map[string]string{topic: newHubLoc})
		if code == 307 {
			log.Printf("Temporary redirect response, to new address {%v}", newHubLoc)
		} else {
			// TODO(adam): Consider replacing old hub url in storage, instead of supplanting
			log.Printf("Permanent redirect response, to new address {%v}", newHubLoc)
		}
		return sub.renewSubscription(ctx, callback)
	}

	return fmt.Errorf("Invalid status code while making subscription request, resp: {%+v}", resp)
}

// helper function to generate a 16-byte (32 chars) string
func generateCallback() string {
	randomURI := make([]byte, 16)
	rand.Read(randomURI)
	return hex.EncodeToString(randomURI)
}
