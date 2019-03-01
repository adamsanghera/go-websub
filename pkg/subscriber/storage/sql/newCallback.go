package sql

import (
	"context"

	sql "database/sql"
)

// NewCallback implies that the client is waiting for reply to a sub request on the given callback
func (sqlStor *SQL) NewCallback(ctx context.Context, topic, hub, callback string) (err error) {
	if topic == "" {
		return ErrMalformedTopic
	}
	if hub == "" {
		return ErrMalformedHub
	}

	tx, err := sqlStor.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: false})
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			tx.Rollback()
		}
	}()

	if _, err := tx.ExecContext(
		ctx, `
		INSERT INTO subscriptions 
		(topic_url, hub_url, callback_url) VALUES 
		(?,?,?)`,
		topic, hub, callback,
	); err != nil {
		return err
	}

	return tx.Commit()
}
