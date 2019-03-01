package sql

import (
	"database/sql"
	"time"

	"github.com/adamsanghera/go-websub/pkg/subscriber/subscriberpb"
)

// GetSubscription returns any subscription associated with the given callback.
func (sqlStor *SQL) GetSubscription(callback string) (*subscriberpb.Subscription, error) {
	row := sqlStor.db.QueryRow(`
		SELECT topic_url, hub_url, callback_url, lease_expiration, lease_initiated, inactive_reason
		FROM active_subscriptions
		WHERE callback_url == ?;`,
		callback,
	)

	var tp, hb, cb, le, li string
	var rea sql.NullString
	if err := row.Scan(&tp, &hb, &cb, &le, &li, &rea); err != nil {
		return nil, err
	}

	exp, err := time.Parse(sqliteTimeFmt, le)
	if err != nil {
		return nil, err
	}

	init, err := time.Parse(sqliteTimeFmt, li)
	if err != nil {
		return nil, err
	}

	return &subscriberpb.Subscription{
		Topic:           tp,
		Hub:             hb,
		Callback:        cb,
		LeaseExpiration: exp.Unix(),
		LeaseInitiated:  init.Unix(),
		InactiveReason:  rea.String,
	}, nil
}
