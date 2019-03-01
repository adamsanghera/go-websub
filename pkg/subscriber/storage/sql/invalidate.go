package sql

import (
	"context"
	"database/sql"
)

// Invalidate is called when a hub no longer views the subscription as active.
// Invalidating is an idempotent action, it is OK to do it more than once.
// Repeated invalidations will return a handleable error, ErrUpdateFailed.
func (sqlStor *SQL) Invalidate(ctx context.Context, callback, inactiveReason string) (err error) {
	if inactiveReason == "" {
		return ErrMalformedInactiveReason
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

	res, err := tx.ExecContext(
		ctx, `
		UPDATE subscriptions
		SET lease_initiated=NULL, lease_expiration=datetime('now'), inactive_reason=?
		WHERE callback_url=? AND (lease_expiration IS NULL OR lease_expiration > datetime('now'));`,
		// TODO(adam) contemplate doing some check here.  Do I even need one?
		inactiveReason, callback,
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
