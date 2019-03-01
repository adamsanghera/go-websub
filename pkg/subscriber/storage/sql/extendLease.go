package sql

import (
	"context"
	"database/sql"
	"time"
)

// ExtendLease provides a subscription lease to a given callback.  Implicitly, this means that the subscription is active.
// This occurs when a subscription is first ACK'd, and also upon subsequent lease renewals.
func (sqlStor *SQL) ExtendLease(ctx context.Context, callback string, newExpiration time.Time) (err error) {
	if newExpiration.Before(time.Now()) {
		return ErrNewLeaseInPast{newExpiration}
	}

	tx, err := sqlStor.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: false})
	if err != nil {
		return err
	}

	// Defer a rollback, if an error is encountered
	defer func() {
		if err != nil {
			tx.Rollback()
		}
	}()

	res, err := tx.ExecContext(ctx, `
		UPDATE subscriptions
		SET lease_expiration=?, lease_initiated=(
			CASE
				WHEN lease_initiated IS NULL
				THEN ?
				ELSE lease_initiated
			END)
		WHERE 
			callback_url=? 
			AND (
				lease_expiration IS NULL
			  OR (
				  lease_expiration IS NOT NULL
			    AND datetime('now') < datetime(lease_expiration)));`,
		newExpiration.UTC().Format(sqliteTimeFmt),
		time.Now().UTC().Format(sqliteTimeFmt),
		callback,
	)
	if err != nil {
		return err
	}

	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n != 1 {
		return ErrUpdateFailed{n}
	}

	return tx.Commit()
}
