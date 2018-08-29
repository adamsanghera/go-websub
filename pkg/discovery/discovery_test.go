package discovery

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"testing"

	"gopkg.in/jarcoal/httpmock.v1"
)

func getBodyFromFile(testCode string) string {
	fname := fmt.Sprintf("test-assets/%s.html", testCode)

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
			resp := httpmock.NewStringResponse(200, getBodyFromFile("100"))
			resp.Header.Set("Link", "<https://hub.example.com/>; rel=\"hub\", <http://example.com/feed>; rel=\"self\"")
			return resp, nil
		})

	t.Run("Parsing Links from comma-delimited link header", func(t *testing.T) {
		hubs, self, err := DiscoverTopic("http://example.com/feed")
		if err != nil {
			t.Error(err)
		}
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
			resp := httpmock.NewStringResponse(200, getBodyFromFile("101"))
			resp.Header.Set("Content-type", "text/html")
			return resp, nil
		})

	t.Run("Parsing Links from html body", func(t *testing.T) {
		hubs, self, err := DiscoverTopic("http://example.com/feed")
		if err != nil {
			t.Error(err)
		}
		if _, ok := hubs["https://websub.rocks/blog/101/kjEJaVI57HetbbiZWivI/hub"]; !ok {
			t.Error("Failed to parse hub link")
		}
		if self != "https://websub.rocks/blog/101/kjEJaVI57HetbbiZWivI" {
			t.Error("Failed to parse self link")
		}
	})
}

func TestMisplacedLinksHTML(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	httpmock.RegisterResponder("GET", "http://example.com/feed",
		func(req *http.Request) (*http.Response, error) {
			resp := httpmock.NewStringResponse(200, getBodyFromFile("101-misplaced"))
			resp.Header.Set("Content-type", "text/html")
			return resp, nil
		})

	// Links are not in the head, so discovery is expected to return
	// with an empty set and string
	t.Run("Fails to find links in the html head", func(t *testing.T) {
		hubs, self, err := DiscoverTopic("http://example.com/feed")
		if err == nil {
			t.Error("Failed to return an error")
		}
		if len(hubs) != 0 {
			t.Error("Found hub links")
		}
		if self != "" {
			t.Error("Found self links")
		}
	})
}

func TestNoHeadHTML(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	httpmock.RegisterResponder("GET", "http://example.com/feed",
		func(req *http.Request) (*http.Response, error) {
			resp := httpmock.NewStringResponse(200, getBodyFromFile("101-no-head"))
			resp.Header.Set("Content-type", "text/html")
			return resp, nil
		})

	// There's no head, so we shouldn't find any links!
	t.Run("Fails to find links in the html head", func(t *testing.T) {
		hubs, self, err := DiscoverTopic("http://example.com/feed")
		if err == nil {
			t.Error("Failed to return an error")
		}
		if len(hubs) != 0 {
			t.Error("Found hub links")
		}
		if self != "" {
			t.Error("Found self links")
		}
	})
}

func TestMalformedHTML(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	httpmock.RegisterResponder("GET", "http://example.com/feed",
		func(req *http.Request) (*http.Response, error) {
			resp := httpmock.NewStringResponse(200, getBodyFromFile("101-malformed"))
			resp.Header.Set("Content-type", "text/html")
			return resp, nil
		})

	// The HTML file itself isn't correct (i.e. can't be parsed)
	// so parsing the file shouldn't find anything
	t.Run("Fails to find links in the html head", func(t *testing.T) {
		hubs, self, err := DiscoverTopic("http://example.com/feed")
		if err == nil {
			t.Error("Failed to return an error")
		}
		if len(hubs) != 0 {
			t.Error("Found hub links")
		}
		if self != "" {
			t.Error("Found self links")
		}
	})
}
