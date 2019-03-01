package storage

import (
	"time"

	"github.com/adamsanghera/go-websub/pkg/subscriber/subscriberpb"
)

// Storage is the root interface for this package, and manages the state of subscription objects.
// For more information, see README.md
type Storage interface {
	/* Commands */

	// IndexOffer indexes the topic <-> hub relationship, which is observed in the discovery phase
	IndexOffer(topicsToHubs map[string]string) error

	// NewCallback records the fact that a subscription with the given hub has been initiated for the given topic.
	NewCallback(topic, hub, callback string) error

	// Invalidate expires a subscription.  This can happen in the cases of hub denials, or user-initiated cancels.
	// Note that the client is NOT expected to invoke this method for subscriptions that merely expire.
	Invalidate(callback, inactiveReason string) error

	// ExtendLease provides a subscription lease to a given callback.  Implicitly, this means that the subscription is active.
	// This occurs when a subscription is first ACK'd, and also upon subsequent lease renewals.
	ExtendLease(callback string, newExpiration time.Time) error

	/* Queries */

	// GetActiveCallback returns a callback if a given topic+hub combination exists
	GetCActiveallback(topic, hub string) (string, error)

	// GetSubscription returns any subscription associated with the given callback.
	GetSubscription(callback string) (*subscriberpb.Subscription, error)

	// GetActive returns at most 'pageSize' active subscriptions in alphabetical order.
	// If there are more than 'pageSize', the caller can use 'pageNum' to ask for a specific partition in the sequence.
	GetActive(pageSize int, lastTopic, lastHub string) (subs *subscriberpb.Subscriptions, lastPage bool, err error)

	// GetInactive returns at most 'pageSize' inactive subscriptions in alphabetical order.
	// If there are more than 'pageSize', the caller can use 'pageNum' to ask for a specific partition in the sequence.
	GetInactive(pageSize int, lastTopic, lastHub string) (subs *subscriberpb.Subscriptions, lastPage bool, err error)
}
