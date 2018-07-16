package discovery

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"testing"

	"gopkg.in/jarcoal/httpmock.v1"
)

func getBodyFromFile(testCode int) string {
	fname := fmt.Sprintf("test-assets/%d.html", testCode)

	if bytes, err := ioutil.ReadFile(fname); err == nil {
		return string(bytes)
	} else {
		panic(err)
	}
}

func TestDiscoverTopic(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	httpmock.RegisterResponder("GET", "http://example.com/feed",
		func(req *http.Request) (*http.Response, error) {
			resp := httpmock.NewStringResponse(200, getBodyFromFile(100))
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
			resp := httpmock.NewStringResponse(200, getBodyFromFile(101))
			resp.Header.Set("Content-type", "text/html")
			return resp, nil
		})

	t.Run("Parsing Links from html body", func(t *testing.T) {
		hubs, self := DiscoverTopic("http://example.com/feed")
		if _, ok := hubs["https://websub.rocks/blog/101/kjEJaVI57HetbbiZWivI/hub"]; !ok {
			t.Error("Failed to parse hub link")
		}
		if self != "https://websub.rocks/blog/101/kjEJaVI57HetbbiZWivI" {
			t.Error("Failed to parse self link")
		}
	})

	/*
		TODO(adam) add tests that have protocol-breaking responses,
		and verify that our protocol doesn't parse them
	*/
}
