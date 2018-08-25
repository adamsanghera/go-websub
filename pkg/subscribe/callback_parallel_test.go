package subscribe

import (
	"strconv"
	"sync"
	"testing"
	"time"

	httpmock "gopkg.in/jarcoal/httpmock.v1"
)

func TestParallelSuccessfulSubscribes(t *testing.T) {
	httpmock.Activate()
	sc := NewServer("4000")

	topics := make([]string, 50)
	callbacks := make([]string, 50)

	for idx := range topics {
		topics[idx] = topicURL + strconv.Itoa(idx)
		registerSuccessfulValidationAck(t, &callbacks[idx], topics[idx])
		sc.topicsToSelf[topics[idx]] = topics[idx]
	}

	t.Run("Parallel successful subscribes", func(t *testing.T) {
		// Create a wait group, so that we can synchronize test exit
		wg := &sync.WaitGroup{}

		for idx, topic := range topics {
			wg.Add(1)
			go func(topic string, callback string, wg *sync.WaitGroup) {
				httpMockMut.Lock()
				sc.subscribe(topic)
				httpMockMut.Unlock()

				sc.pSubsMut.Lock()
				if _, ok := sc.pendingSubs[topic]; !ok {
					t.Fatal("Subscription not registered as pending")
				}
				sc.pSubsMut.Unlock()

				// Hit the callback with a successful verification
				verifyCallback(t, topic, callback, "subscribe")

				sc.aSubsMut.Lock()
				if _, exists := sc.activeSubs[topic]; !exists {
					t.Fatal("Subscription is not in active set, even though it was accepted")
				}
				sc.aSubsMut.Unlock()

				sc.pSubsMut.Lock()
				if _, exists := sc.pendingSubs[topic]; exists {
					t.Fatal("Subscription is in pending set, even though it was accepted")
				}
				sc.pSubsMut.Unlock()
				wg.Done()
			}(topic, callbacks[idx], wg)
		}

		// Wait for all subscription routines to finish
		wg.Wait()
	})

	sc.Shutdown()
	httpmock.DeactivateAndReset()
}

func TestParallelSubscribesWithSomeDenials(t *testing.T) {
	httpmock.Activate()
	sc := NewServer("4000")

	topics := make([]string, 50)
	callbacks := make([]string, 50)

	for idx := range topics {
		topics[idx] = topicURL + strconv.Itoa(idx)
		registerSuccessfulValidationAck(t, &callbacks[idx], topics[idx])
		sc.topicsToSelf[topics[idx]] = topics[idx]
	}

	t.Run("Parallel Subs with some Denials", func(t *testing.T) {
		// Create a wait group, so that we can synchronize test exit
		wg := &sync.WaitGroup{}

		for idx, topic := range topics {
			wg.Add(1)
			go func(topic string, callback string, idx int, wg *sync.WaitGroup) {
				httpMockMut.Lock()
				sc.subscribe(topic)
				httpMockMut.Unlock()

				sc.pSubsMut.Lock()
				if _, ok := sc.pendingSubs[topic]; !ok {
					t.Fatal("Subscription not registered as pending")
				}
				sc.pSubsMut.Unlock()

				// Hit the callback with a successful verification
				if idx%3 == 0 {
					denyCallback(t, topic, callback)
					sc.aSubsMut.Lock()
					if _, exists := sc.activeSubs[topic]; exists {
						t.Fatal("Subscription is in active set, even though it was denied")
					}
					sc.aSubsMut.Unlock()
					sc.pSubsMut.Lock()
					if _, exists := sc.pendingSubs[topic]; exists {
						t.Fatal("Subscription is in pending set, even though it was denied")
					}
					sc.pSubsMut.Unlock()
				} else {
					verifyCallback(t, topic, callback, "subscribe")
					sc.aSubsMut.Lock()
					if _, exists := sc.activeSubs[topic]; !exists {
						t.Fatal("Subscription is not in active set, even though it was accepted")
					}
					sc.aSubsMut.Unlock()
					sc.pSubsMut.Lock()
					if _, exists := sc.pendingSubs[topic]; exists {
						t.Fatal("Subscription is in pending set, even though it was accepted")
					}
					sc.pSubsMut.Unlock()
				}

				wg.Done()
			}(topic, callbacks[idx], idx, wg)
		}

		// Wait for all subscription routines to finish
		wg.Wait()

		// Give everything time to settle
		time.Sleep(50 * time.Millisecond)

		for idx, topic := range topics {
			if idx%3 == 0 {
				sc.aSubsMut.Lock()
				if _, exists := sc.activeSubs[topic]; exists {
					t.Fatal("Subscription is in active set, even though it was denied")
				}
				sc.aSubsMut.Unlock()
				sc.pSubsMut.Lock()
				if _, exists := sc.pendingSubs[topic]; exists {
					t.Fatal("Subscription is in pending set, even though it was denied")
				}
				sc.pSubsMut.Unlock()
			} else {
				sc.aSubsMut.Lock()
				if _, exists := sc.activeSubs[topic]; !exists {
					t.Fatal("Subscription is not in active set, even though it was accepted")
				}
				sc.aSubsMut.Unlock()
				sc.pSubsMut.Lock()
				if _, exists := sc.pendingSubs[topic]; exists {
					t.Fatal("Subscription is in pending set, even though it was accepted")
				}
				sc.pSubsMut.Unlock()
			}
		}
	})

	sc.Shutdown()
	httpmock.DeactivateAndReset()
}
