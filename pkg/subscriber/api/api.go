package api

// API is the websub api interface, abstracted away from HTTP
type API interface {
	Discover(topic string) error       // Discover new topics!
	Subscribe(topic, hub string) error // Send a subscription request
	ReceiveCallback() <-chan string    // Handle callback
}
