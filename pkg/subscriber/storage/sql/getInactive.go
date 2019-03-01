package sql

import (
	"database/sql"

	"github.com/adamsanghera/go-websub/pkg/subscriber/subscriberpb"
)

// GetInactive returns at most 'pageSize' inactive topic/hub tuples in alphabetical order.
// If there are more than 'pageSize', the caller can use 'lastTopic' to ask for the next 'pageSize' topic/hub tuples.
func (sqlStor *SQL) GetInactive(pageSize int, lastTopic, lastHub string) (subs *subscriberpb.Subscriptions, lastPage bool, err error) {
	rows, err := sqlStor.db.Query(`
		SELECT topic_url, hub_url, callback_url
		FROM inactive_subscriptions
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

	// Read our rows
	rowCtr := 0
	for rows.Next() {
		var topic, hub string
		var callback sql.NullString
		err = rows.Scan(&topic, &hub, &callback)
		if err != nil {
			return nil, false, err
		}

		// Add info to list
		subs.Subscriptions = append(
			subs.Subscriptions,
			&subscriberpb.Subscription{
				Topic:    topic,
				Hub:      hub,
				Callback: callback.String,
			},
		)
		rowCtr++
	}

	return subs, rowCtr < pageSize, nil
}
