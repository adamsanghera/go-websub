package subscriber

import (
	"github.com/adamsanghera/go-websub/pkg/discovery"
)

// DiscoverTopic runs the common discovery algorithm, and indexes the results
func (sc *Subscriber) DiscoverTopic(topic string) error {
	hubs, self, err := discovery.DiscoverTopic(topic)
	if err != nil {
		return err
	}

	// NOTE(adam) consider rewriting DiscoverTopic or SubscriptionAddOffer
	topicsToHubs := make(map[string]string)
	for h := range hubs {
		topicsToHubs[self] = h
	}

	return sc.storage.IndexOffer(topicsToHubs)
}
