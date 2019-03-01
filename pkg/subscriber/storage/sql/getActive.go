package sql

import (
	"time"

	"github.com/adamsanghera/go-websub/pkg/subscriber/subscriberpb"
)

// GetActive returns at most 'pageSize' active subscriptions in alphabetical order.
// If there are more than 'pageSize', the caller can use 'pageNum' to ask for a specific partition in the sequence.
func (sql *SQL) GetActive(pageSize int, lastTopic, lastHub string) (subs *subscriberpb.Subscriptions, lastPage bool, err error) {
	rows, err := sql.db.Query(`
		SELECT topic_url, hub_url, callback_url, lease_initiated
		FROM active_subscriptions
		WHERE (topic_url, hub_url) > (?, ?)
		ORDER BY topic_url, hub_url
		LIMIT ?;`,
		lastTopic,
		lastHub,
		pageSize,
	)
	defer rows.Close()
	if err != nil {
		return nil, false, err
	}

	subs = &subscriberpb.Subscriptions{}

	rowCtr := 0
	for rows.Next() {
		var topic, hub, callback, leaseInitiated string
		err = rows.Scan(&topic, &hub, &callback, &leaseInitiated)
		if err != nil {
			return nil, false, err
		}

		lease, err := time.Parse(sqliteTimeFmt, leaseInitiated)
		if err != nil {
			return nil, false, ErrMalformedTime{leaseInitiated}
		}

		// Add info to list
		subs.Subscriptions = append(
			subs.Subscriptions,
			&subscriberpb.Subscription{
				Callback:       callback,
				Topic:          topic,
				Hub:            hub,
				LeaseInitiated: lease.Unix(),
			},
		)
		rowCtr++
	}

	return subs, rowCtr < pageSize, nil
}
