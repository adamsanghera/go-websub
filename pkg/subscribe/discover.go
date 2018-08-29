package subscribe

import "github.com/adamsanghera/go-websub/pkg/discovery"

// DiscoverTopic runs the common discovery algorithm, and compiles its results into the Server map
func (sc *Server) DiscoverTopic(topic string) error {
	hubs, self, err := discovery.DiscoverTopic(topic)

	if err != nil {
		return err
	}

	sc.tthMut.Lock()
	defer sc.tthMut.Unlock()
	sc.ttsMut.Lock()
	defer sc.ttsMut.Unlock()

	// Allocate the map if necessary
	if _, ok := sc.topicsToHubs[topic]; !ok {
		sc.topicsToHubs[topic] = make(map[string]struct{})
	}

	// Iterate through the results
	for hub := range hubs {
		sc.topicsToHubs[topic][hub] = struct{}{}
	}
	sc.topicsToSelf[topic] = self

	return nil
}
