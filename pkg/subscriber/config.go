package subscriber

// Config is the configuration information for a Subscriber
type Config struct {
	port string
}

// NewConfig returns the default config for Subscriber
func NewConfig() *Config {
	return &Config{
		"4000",
	}
}
