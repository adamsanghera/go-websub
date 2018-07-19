package subscriber

import (
	"log"
	"net/http"
	"strconv"
	"time"
)

// Callback is the function that is hit when a hub responds to
// a sub/un-sub request.
func (sc *Client) Callback(w http.ResponseWriter, req *http.Request) {
	// Differentiate between verification and denial notifications
	query := req.URL.Query()
	switch query.Get("hub.mode") {
	case "denied":
		topic := query.Get("hub.topic")
		reason := query.Get("hub.reason")
		log.Printf("Subscription to topic %s from url %s rejected.  Reason provided: {%s}", topic, req.Host, reason)
	case "subscribe":
		sc.handleAckedSubscription(w, req)
	case "unsubscribe":
		topic := query.Get("hub.topic")
		challenge := query.Get("hub.challenge")
		log.Printf("Unsubscribe from topic %s from url %s verification begin.  Challenge provided: {%s}.", topic, req.Host, challenge)

		if _, exists := sc.pendingUnSubs[topic]; exists {
			w.WriteHeader(200)
			w.Write([]byte(challenge))
			delete(sc.pendingSubs, topic)
			return
		}

		// Received a callback for a function that we did not send
		w.WriteHeader(404)
		w.Write([]byte(""))
	default:
		// This indicates a broken request.
	}
}

// This handler performs the following actions:
// 1. Checks if the subscription is pending (reject ACK if not)
// 2. Spawns a routine that will renew the subscription 2/3 of the way through a lease
// 3. ACKs the subscription ACK, by writing the challenge back with a 200 code
// 4. Removes the subscription from the pending set
// 5. Adds the subscription to the active set
func (sc *Client) handleAckedSubscription(w http.ResponseWriter, req *http.Request) {
	query := req.URL.Query()

	topic := query.Get("hub.topic")
	challenge := query.Get("hub.challenge")
	leaseSeconds := query.Get("hub.lease_seconds")
	log.Printf("Subscription to topic %s from url %s verification begin.  Challenge provided: {%s}.  Lease length (s): {%s}", topic, req.Host, challenge, leaseSeconds)

	// 1
	if _, exists := sc.pendingSubs[topic]; exists {

		// 2
		go func() {
			seconds, err := strconv.Atoi(leaseSeconds)
			if err != nil {
				panic(err)
			}
			// Sleep for some proportion of the lease time
			time.Sleep(time.Duration(2*seconds/3) * time.Second)

			// If there's an error, log and delete the topic from subscribed list
			if err = sc.SubscribeToTopic(topic); err != nil {
				log.Printf("Encountered an error while renewing subscription {%v}", err)
				delete(sc.activeSubs, topic)
			}
		}()

		// 3
		w.WriteHeader(200)
		w.Write([]byte(challenge))

		// 4
		delete(sc.pendingSubs, topic)

		// 5
		if _, allocated := sc.activeSubs[topic]; !allocated {
			sc.activeSubs[topic] = struct{}{}
		}

		return
	}

	// (1)
	w.WriteHeader(404)
	w.Write([]byte(""))

}
