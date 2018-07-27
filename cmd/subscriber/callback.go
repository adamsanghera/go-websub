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

// CallbackSwitch is the branching point between the various types of callback responses.
// According to the specification, /callback/* is only hit under the following circumstances:
// 1. Verifying a subscription request [5.3]
// 2. Verifying an unsubscription request [5.3]
// 3. Denying a subscription request (even after it has been accepted) [5.2]
func (sc *Client) CallbackSwitch(w http.ResponseWriter, req *http.Request) {
	endpoint := strings.Split(req.URL.Path, "/callback/")[1]
	reqBody, _ := ioutil.ReadAll(req.Body)
	query, _ := url.ParseQuery(string(reqBody))

	// TODO(adam)
	// Currently, we want to "stop the world" of subscriptions for the duration of this handler, just to be safe.
	// However! Once this section of the codebase settles down, we can look at maybe taking a more fine-grained approach.
	sc.pSubsMut.Lock()
	sc.aSubsMut.Lock()
	sc.pUnSubsMut.Lock()

	defer sc.aSubsMut.Unlock()
	defer sc.pSubsMut.Unlock()
	defer sc.pUnSubsMut.Unlock()

	switch query.Get("hub.mode") {
	case "subscribe":
		sc.handleSubscription(w, endpoint, query)
	case "denied":
		sc.handleDeniedSubscription(query)
	case "unsubscribe":
		sc.handleUnsubscription(w, endpoint, query)
	default:
		// NOTE(adam): should we write back?  Spec makes no recommendation.
	}

}

// Handles subscription events
// Stated below are the possible sub states at the moment this function is called
// 1. Sub pending, sub active
// 2. Sub pending, sub NOT active
// 3. Sub NOT pending, sub active
// 4. Sub NOT pending, sub NOT active
func (sc *Client) handleSubscription(w http.ResponseWriter, endpoint string, query url.Values) {
	topic := query.Get("hub.topic")
	challenge := query.Get("hub.challenge")
	leaseSeconds := query.Get("hub.lease_seconds")
	log.Printf("Verifying sub to topic %s.  Challenge provided: {%s}.  Lease length (s): {%s}\n", topic, challenge, leaseSeconds)

	if _, isPending := sc.pendingSubs[topic]; isPending {
		// 1 and 2 are handled in the same way
		seconds, err := strconv.Atoi(leaseSeconds)
		if err != nil {
			panic(err)
		}
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*time.Duration(seconds*2/3))
		sc.activeSubs[topic] = &activeSubscription{
			callbackURL: endpoint,
			cancel:      cancel,
		}

		go sc.renewSubscription(ctx, topic)

		delete(sc.pendingSubs, topic)
		sc.activeSubs[topic] = &activeSubscription{
			callbackURL: endpoint,
			cancel:      cancel,
		}

		// From 5.3.1: The subscriber MUST respond with an 2xx code,
		//   with response body equal to the hub.challenge param.
		w.WriteHeader(200)
		w.Write([]byte(challenge))
	} else {
		// From 5.3.1:
		//   If the subscriber does not agree with the action, the subscriber MUST respond with a 404.
		// Here, disagreement is defined as receiving verification request for a subscription that is not pending.
		w.WriteHeader(404)
		w.Write([]byte(""))
	}

}

// Handles denial events.
// Stated below are the possible sub states at the moment this function is called
// 1. Sub pending, active		  [denied while renewing an active sub]
// 2. Sub pending, NOT active     [denied before first approval is given]
// 3. Sub NOT pending, active     [NOTE(adam): unclear whether this is in spec or not]
// 4. Sub NOT pending, NOT active [possibly duped denial request]
func (sc *Client) handleDeniedSubscription(query url.Values) {
	topic := query.Get("hub.topic")
	reason := query.Get("hub.reason")
	log.Printf("Subscription to topic %s rejected.  Reason provided: {%s}", topic, reason)

	if _, isPending := sc.pendingSubs[topic]; isPending {
		if sub, isActive := sc.activeSubs[topic]; isActive {
			// 1
			sub.cancel()
			delete(sc.activeSubs, topic)
		} else {
			// 2: no-op
		}
		delete(sc.pendingSubs, topic)
	} else {
		if sub, isActive := sc.activeSubs[topic]; isActive {
			// 3: NOTE(adam): Unclear if this flow is in-spec.
			// I'm going to play it safe and fail closed (deleting the active subscription).
			// In this case, the client will assume that this sub is no longer active, and it's
			//   up to the wrapper application to trigger another subscription.
			sub.cancel()
			delete(sc.activeSubs, topic)
			log.Println("DENIED anomaly: sub not pending, sub active. Spec unclear. Hub is probably to blame.")
		} else {
			// 4: no-op
			log.Println("DENIED anomaly: sub not pending, sub not active. Hub thinks sub is active, but client disagrees.")
		}
	}

	// NOTE(adam): Spec is unclear whether hub is listening for a response to this GET.
	// I'm just going to let the response writer die.
}

// Handles unsubscription events.
// Stated below are the possible sub states at the moment this function is called
// 1 Unsub pending, sub active          [normal case]
// 2 Unsub pending, sub NOT active      [unsub requested, after sub was momentarily active but later denied]
// 3 Unsub NOT pending, sub active      [hub accidentally duplicated unsub response]
// 4 Unsub NOT pending, sub NOT active  [commander accidentally unsubbed twice]
//
// NOTE(adam): This handler does NOT touch *or even check* pendingSubs.
func (sc *Client) handleUnsubscription(w http.ResponseWriter, endpoint string, query url.Values) {
	topic := query.Get("hub.topic")
	challenge := query.Get("hub.challenge")

	if callback, exists := sc.pendingUnSubs[topic]; exists && callback == endpoint {
		if sub, exists := sc.activeSubs[topic]; exists {
			// 1
			sub.cancel()
			delete(sc.activeSubs, topic)
		} else {
			// 2 This isn't necessarily an error, but it is abnormal behavior, worth logging
			log.Println("UNSUB anomaly: unsub pending, sub not active. Within spec, but rare. Buy a lotto ticket.")
		}
		delete(sc.pendingSubs, topic)

		// From 5.3.1: The subscriber MUST respond with an 2xx code,
		//   with response body equal to the hub.challenge param.
		w.WriteHeader(200)
		w.Write([]byte(challenge))
	} else {
		// From 5.3.1:
		//   The subscriber MUST confirm that the hub.topic corresponds to a pending subscription or unsubscription that
		//   the subscriber wishes to carry out. If so, the subscriber MUST respond with an HTTP success (2xx) code with
		//   a response body equal to the hub.challenge parameter.
		//
		//   If the subscriber does not agree with the action, the subscriber MUST respond with a 404 "Not Found" response.
		//
		// Accordingly, we should not mutate any client state here, just spit back a 404.
		w.WriteHeader(404)
		w.Write([]byte(challenge))

		log.Println("UNSUB anomaly: unsub not pending, sub not active. Hub thinks client is requesting unsub, client disagrees.")
	}
}

// Routine that spends 99% of its life asleep, waiting to be cancelled or start a sub request.
func (sc *Client) renewSubscription(ctx context.Context, topic string) {
	// Block until cancelled or timeout
	<-ctx.Done()

	sc.aSubsMut.Lock()
	defer sc.aSubsMut.Unlock()

	// If the context was cancelled, just die gracefully.
	// NOTE(adam): This function trusts that the canceller handles clean-up of active subscriptions
	if ctx.Err() == context.Canceled {
		return
	}

	// Otherwise, keep on trying to subscribe
	if err := sc.Subscribe(topic); err != nil {
		// If there's an error, log it
		log.Printf("Encountered an error while renewing subscription {%v}", err)
	}
}
