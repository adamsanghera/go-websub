package sql

// GetActiveCallback returns a callback if a given topic+hub combination exists
func (sql *SQL) GetActiveCallback(topic, hub string) (string, error) {
	row := sql.db.QueryRow(`
		SELECT callback_url
		FROM active_subscriptions
		WHERE topic_url == ? AND hub_url == ?;`,
		topic,
		hub,
	)

	var callback string
	if err := row.Scan(&callback); err != nil {
		return "", err
	}

	return callback, nil
}
