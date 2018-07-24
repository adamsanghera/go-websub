package subscriber

import (
	"context"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// CallbackSwitch is the branching point between the various types of callback responses
func (sc *Client) CallbackSwitch(w http.ResponseWriter, req *http.Request) {
	endpoint := strings.Split(req.URL.Path, "/callback/")[1]
	reqBody, _ := ioutil.ReadAll(req.Body)
	query, _ := url.ParseQuery(string(reqBody))

	if callbackURL, exists := sc.pendingSubs[query.Get("hub.topic")]; exists && callbackURL == endpoint {
		switch query.Get("hub.mode") {
		case "denied":
			sc.handleDeniedSubscription(w, query)
		case "subscribe":
			sc.handleSubscription(w, query)
		case "unsubscribe":
			sc.handleUnsubscription(w, query)
		default:
			// This indicates a broken request.
		}
	} else {
		w.WriteHeader(404)
		w.Write([]byte(""))
	}
}

// This handler performs the following actions:
// 1. Checks if the subscription is pending (reject ACK if not)
// 2. Spawns a routine that will renew the subscription 2/3 of the way through a lease
// 3. ACKs the subscription ACK, by writing the challenge back with a 200 code
// 4. Removes the subscription from the pending set
// 5. Adds the subscription to the active set
func (sc *Client) handleSubscription(w http.ResponseWriter, query url.Values) {
	topic := query.Get("hub.topic")
	challenge := query.Get("hub.challenge")
	leaseSeconds := query.Get("hub.lease_seconds")
	log.Printf("Verifying sub to topic %s.  Challenge provided: {%s}.  Lease length (s): {%s}", topic, challenge, leaseSeconds)

	sc.pSubsMut.Lock()
	defer sc.pSubsMut.Unlock()

	// 1
	if _, exists := sc.pendingSubs[topic]; exists {

		sc.aSubsMut.Lock()
		defer sc.aSubsMut.Unlock()

		// 2
		seconds, err := strconv.Atoi(leaseSeconds)
		if err != nil {
			panic(err)
		}
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*time.Duration(seconds*2/3))

		go sc.stickySubscription(ctx, topic)

		// 3
		w.WriteHeader(200)
		w.Write([]byte(challenge))

		// 4
		delete(sc.pendingSubs, topic)

		// 5
		sc.activeSubs[topic] = cancel

		return
	}

	// (1)
	w.WriteHeader(404)
	w.Write([]byte(""))

}

func (sc *Client) handleDeniedSubscription(w http.ResponseWriter, query url.Values) {
	sc.pSubsMut.Lock()
	defer sc.pSubsMut.Unlock()

	sc.aSubsMut.Lock()
	defer sc.aSubsMut.Unlock()

	topic := query.Get("hub.topic")
	reason := query.Get("hub.reason")

	log.Printf("Subscription to topic %s rejected.  Reason provided: {%s}", topic, reason)

	// Case where sub is pending
	if _, exists := sc.pendingSubs[topic]; exists {
		delete(sc.pendingSubs, topic)
	}

	// Case where sub is already active
	if _, exists := sc.activeSubs[topic]; exists {
		delete(sc.activeSubs, topic)
	}

	w.WriteHeader(200)
	w.Write([]byte(""))
}

func (sc *Client) handleUnsubscription(w http.ResponseWriter, query url.Values) {
	sc.pUnSubsMut.Lock()
	defer sc.pUnSubsMut.Unlock()
	sc.aSubsMut.Lock()
	defer sc.aSubsMut.Unlock()

	topic := query.Get("hub.topic")
	challenge := query.Get("hub.challenge")

	log.Printf("Verifying unsub from topic %s.  Challenge provided: {%s}.", topic, challenge)

	// Remove from pending (if it exists)
	if _, exists := sc.pendingUnSubs[topic]; exists {
		w.WriteHeader(200)
		w.Write([]byte(challenge))
		delete(sc.pendingSubs, topic)
		return
	}

	// Call the cancel function
	if cancelFunc, exists := sc.activeSubs[topic]; exists {
		// Need to unlock this, so that stickySubscription can acquire
		sc.aSubsMut.Unlock()
		cancelFunc()
	}
}

func (sc *Client) stickySubscription(ctx context.Context, topic string) {
	// Block until cancelled or deadline
	<-ctx.Done()

	sc.aSubsMut.Lock()
	defer sc.aSubsMut.Unlock()

	// If the context was cancelled, just die gracefully, and remove the active sub
	if ctx.Err() == context.Canceled {
		delete(sc.activeSubs, topic)
		return
	}

	// Otherwise, keep on trying to subscribe
	if err := sc.Subscribe(topic); err != nil {
		// If there's an error, log and delete the topic from subscribed list
		log.Printf("Encountered an error while renewing subscription {%v}", err)
		delete(sc.activeSubs, topic)
	}
}
