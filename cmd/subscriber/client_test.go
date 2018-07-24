package subscriber

import (
	"net/http"
	"testing"

	httpmock "gopkg.in/jarcoal/httpmock.v1"
)

func TestClient_SuccessfulSubscription(t *testing.T) {
	sc := NewClient("4000")

	// POSTing to this url will always result in a 202 (positive ack!)
	httpmock.Activate()
	httpmock.RegisterResponder("POST", "http://example.com/feed",
		func(req *http.Request) (*http.Response, error) {
			resp := httpmock.NewStringResponse(202, "")
			return resp, nil
		})

	t.Run("Successful subscription", func(t *testing.T) {
		sc.topicsToSelf["http://example.com/feed"] = "http://example.com/feed"
		err := sc.Subscribe("http://example.com/feed")
		if err != nil {
			t.Error("Failed to subscribe", err)
		}
	})

	sc.Shutdown()
	httpmock.DeactivateAndReset()
}

// TODO(adam) tests that target parallelism in SubscribeToTopic
// Particularly interested in pushing redirect chains:
//   x --> y --> x --> y
