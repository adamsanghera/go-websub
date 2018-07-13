package common

import (
	"net/http"
	"testing"

	"gopkg.in/jarcoal/httpmock.v1"
)

func TestDiscoverTopic(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	httpmock.RegisterResponder("GET", "http://example.com/feed",
		func(req *http.Request) (*http.Response, error) {
			resp := httpmock.NewStringResponse(400, "")
			resp.Header.Set("Link", "<https://hub.example.com/>; rel=\"hub\", <http://example.com/feed>; rel=\"self\"")
			return resp, nil
		})

	type args struct {
		topic string
	}
	tests := []struct {
		name string
	}{
		struct {
			name string
		}{
			"hi",
		},
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hubs, self := DiscoverTopic("http://example.com/feed")
			if _, ok := hubs["https://hub.example.com/"]; !ok {
				t.Error("Failed to parse hub link from comma-delimited link header")
			}
			if self != "http://example.com/feed" {
				t.Error("Failed to parse self link from comma-delimited link header")
			}
		})
	}
}
