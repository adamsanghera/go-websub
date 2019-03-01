package subscriber

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// callbackSwitch is the branching point between the various types of callback responses.
func (sub *Subscriber) callbackSwitch(w http.ResponseWriter, req *http.Request) {
	endpoint := strings.Split(req.URL.Path, "/callback/")[1]
	reqBody, _ := ioutil.ReadAll(req.Body)
	query, _ := url.ParseQuery(string(reqBody))

	err := sub.updateSubscription(query, endpoint)
	if err != nil {
		log.Printf("Encountered error while updating subscription %v: %v\n", endpoint, err)
		w.WriteHeader(404)
		w.Write([]byte(err.Error()))
		return
	}

	// Write response
	challenge := query.Get("hub.challenge")
	w.WriteHeader(200)
	w.Write([]byte(challenge))
}

func (sub *Subscriber) updateSubscription(query url.Values, endpoint string) error {
	action := query.Get("hub.mode")
	if action == "subscribe" {
		seconds, err := time.ParseDuration(query.Get("hub.lease_seconds") + "s")
		if err != nil {
			return err
		}

		err = sub.storage.ExtendLease(context.Background(), endpoint, time.Now().Add(seconds))
		if err != nil {
			return err
		}
		sub.launchRenewal(endpoint, seconds)
	} else if action == "unsubscribe" || action == "denied" {
		return sub.storage.Invalidate(context.Background(), endpoint, action+": "+query.Get("hub.reason"))
	} else {
		return fmt.Errorf("request on /callback {%s} lacked an appropriate hub.mode parameter", endpoint)
	}

	return nil
}

// Tries to renew a subscription after 1/3 of the lease duration has expired.
func (sub *Subscriber) launchRenewal(callback string, leaseSeconds time.Duration) {
	renewalContext, cancel := context.WithTimeout(context.Background(), leaseSeconds)

	go time.AfterFunc(leaseSeconds*1/3, func() {
		defer cancel()
		err := sub.renewSubscription(renewalContext, callback)
		if err == context.Canceled {
			// TODO(adam):
		}
		if err == context.DeadlineExceeded {
			// TODO(adam):
		}
	})
}
