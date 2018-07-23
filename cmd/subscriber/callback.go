package subscriber

import (
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
		// 2

		sc.aSubsMut.Lock()
		defer sc.aSubsMut.Unlock()

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

func (sc *Client) handleDeniedSubscription(w http.ResponseWriter, query url.Values) {
	sc.pSubsMut.Lock()
	defer sc.pSubsMut.Unlock()

	topic := query.Get("hub.topic")
	reason := query.Get("hub.reason")
	log.Printf("Subscription to topic %s rejected.  Reason provided: {%s}", topic, reason)
	delete(sc.pendingSubs, topic)
}

func (sc *Client) handleUnsubscription(w http.ResponseWriter, query url.Values) {
	sc.pUnSubsMut.Lock()
	defer sc.pUnSubsMut.Unlock()
	sc.aSubsMut.Lock()
	defer sc.aSubsMut.Unlock()

	topic := query.Get("hub.topic")
	challenge := query.Get("hub.challenge")

	log.Printf("Verifying unsub from topic %s.  Challenge provided: {%s}.", topic, challenge)

	if _, exists := sc.pendingUnSubs[topic]; exists {
		w.WriteHeader(200)
		w.Write([]byte(challenge))
		delete(sc.pendingSubs, topic)
		return
	}

	if _, exists := sc.activeSubs[topic]; exists {
		delete(sc.activeSubs, topic)
	}
}
