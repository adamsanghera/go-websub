package subscriber

import (
	"net/http"
	"testing"

	httpmock "gopkg.in/jarcoal/httpmock.v1"
)

func TestClient_SubscribeToTopic(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	httpmock.RegisterResponder("POST", "http://example.com/feed",
		func(req *http.Request) (*http.Response, error) {
			resp := httpmock.NewStringResponse(202, "")
			return resp, nil
		})

	t.Run("Successful subscription", func(t *testing.T) {
		sc := NewClient("localhost:4000")
		sc.topicsToSelf["http://example.com/feed"] = "http://example.com/feed"
		err := sc.SubscribeToTopic("http://example.com/feed")
		if err != nil {
			t.Error("Failed to subscribe", err)
		}
	})

	/*
		Todo(adam) tests for
		1. redirects
		   1. temporary
		   2. perm
		2. no url exists for topic
		3. invalid status response code
	*/
}
