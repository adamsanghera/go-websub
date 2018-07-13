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
			resp := httpmock.NewStringResponse(200, "")
			resp.Header.Set("Link", "<https://hub.example.com/>; rel=\"hub\", <http://example.com/feed>; rel=\"self\"")
			return resp, nil
		})

	t.Run("Parsing Links from comma-delimited link header", func(t *testing.T) {
		hubs, self := DiscoverTopic("http://example.com/feed")
		if _, ok := hubs["https://hub.example.com/"]; !ok {
			t.Error("Failed to parse hub link")
		}
		if self != "http://example.com/feed" {
			t.Error("Failed to parse self link")
		}
	})

	httpmock.Reset()
	httpmock.RegisterResponder("GET", "http://example.com/feed",
		func(req *http.Request) (*http.Response, error) {
			resp := httpmock.NewStringResponse(200, "<!doctype html>\n<html>\n<head>\n<link rel=\"hub\" href=\"https://hub.example.com/\">\n<link rel=\"self\" href=\"http://example.com/feed\">\n</head>\n<body>\n...\n</body>\n</html>")
			resp.Header.Set("Content-type", "text/html")
			return resp, nil
		})

	t.Run("Parsing Links from html body", func(t *testing.T) {
		hubs, self := DiscoverTopic("http://example.com/feed")
		if _, ok := hubs["https://hub.example.com/"]; !ok {
			t.Error("Failed to parse hub link")
		}
		if self != "http://example.com/feed" {
			t.Error("Failed to parse self link")
		}
	})
}
