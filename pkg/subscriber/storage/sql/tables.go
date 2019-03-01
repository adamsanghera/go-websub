package sql

const (
	sqliteTimeFmt     = "2006-01-02 15:04:05"
	subscriptionTable = `
		CREATE TABLE IF NOT EXISTS subscriptions (
			topic_url TEXT NOT NULL,
			hub_url TEXT NOT NULL,
			callback_url TEXT NOT NULL,
			lease_expiration TEXT DEFAULT NULL,
			lease_initiated TEXT DEFAULT NULL,
			inactive_reason TEXT DEFAULT NULL,
			
			CHECK (
				lease_expiration IS NULL
				OR datetime(lease_expiration) > datetime (lease_initiated)
			),
			UNIQUE (topic_url, hub_url) ON CONFLICT REPLACE,
			FOREIGN KEY (topic_url, hub_url) REFERENCES offered_subscriptions (topic_url, hub_url),
			PRIMARY KEY (callback_url));`

	offeredSubscriptionsTable = `
		CREATE TABLE IF NOT EXISTS offered_subscriptions (
			topic_url TEXT NOT NULL,
			hub_url TEXT NOT NULL,
		
			PRIMARY KEY (topic_url, hub_url));`

	activeView = `
		CREATE VIEW IF NOT EXISTS active_subscriptions (
			topic_url, hub_url, callback_url, lease_expiration, lease_initiated, inactive_reason
		) AS 
		SELECT topic_url, hub_url, callback_url, lease_expiration, lease_initiated, inactive_reason
		FROM subscriptions
		WHERE ( 
		  lease_expiration IS NOT NULL
			AND datetime('now') < datetime(lease_expiration))
		ORDER BY topic_url, hub_url;`

	inactiveView = `
		CREATE VIEW IF NOT EXISTS inactive_subscriptions (
			topic_url, hub_url, callback_url, inactive_reason
		) AS
		SELECT topic_url, hub_url, callback_url, inactive_reason
		FROM offered_subscriptions
		LEFT OUTER JOIN subscriptions
		USING (topic_url, hub_url)
		WHERE (
			lease_expiration IS NULL
			OR (
				lease_expiration IS NOT NULL 
				AND datetime(lease_expiration) < datetime('now')))
		ORDER BY topic_url, hub_url;`
)
