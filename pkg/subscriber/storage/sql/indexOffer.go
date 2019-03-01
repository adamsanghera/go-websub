package sql

// IndexOffer indexes the topic <-> hub relationship, which is observed in the discovery phase
func (strg *SQL) IndexOffer(topicsToHubs map[string]string) (err error) {
	tx, err := strg.db.Begin()
	if err != nil {
		return err
	}

	// Defer a rollback, if an error is encountered
	defer func() {
		if err != nil {
			tx.Rollback()
		}
	}()

	stmt, err := tx.Prepare(`
	INSERT OR IGNORE INTO offered_subscriptions (
		topic_url, hub_url
	)
	VALUES (
		?,?
	);`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for topic, hub := range topicsToHubs {
		// TODO(adam) better validation
		if topic == "" {
			return ErrMalformedTopic
		}
		if hub == "" {
			return ErrMalformedHub
		}
		if _, err := stmt.Exec(topic, hub); err != nil {
			return err
		}
	}
	return tx.Commit()
}
