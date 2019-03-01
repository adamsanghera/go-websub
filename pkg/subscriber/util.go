package subscriber

import (
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"testing"

	httpmock "gopkg.in/jarcoal/httpmock.v1"
)

func setupDummyValidationAck(hubURL string) {
	httpmock.RegisterResponder("POST", hubURL,
		func(req *http.Request) (*http.Response, error) {
			return httpmock.NewStringResponse(202, ""), nil
		})
}

func setupTempRedirect(hubURL, redirectURL string, t *testing.T, testExp func(req *http.Request) error) {
	httpmock.RegisterResponder("POST", hubURL,
		func(req *http.Request) (*http.Response, error) {
			if err := testExp(req); err != nil {
				t.Fatal(err)
			}
			resp := httpmock.NewStringResponse(307, "")
			resp.Header.Set("Location", redirectURL)
			return resp, nil
		})
}

func setupPermRedirect(hubURL, redirectURL string, t *testing.T, testExp func(req *http.Request) error) {
	httpmock.RegisterResponder("POST", hubURL,
		func(req *http.Request) (*http.Response, error) {
			if err := testExp(req); err != nil {
				t.Fatal(err)
			}
			resp := httpmock.NewStringResponse(308, "")
			resp.Header.Set("Location", redirectURL)
			return resp, nil
		})
}

// verifyCallback will send a verification POST to the given callback.
// It expects to receive a parrotted challenge in response.
// httpmock is deactivated initially (to hit the callback), and reactivated on exit.
func verifyCallback(t *testing.T, topicURL string, callback string, mode string) {
	// Message body
	data := make(url.Values)
	data.Set("hub.mode", mode)
	data.Set("hub.topic", topicURL)
	data.Set("hub.challenge", "kitties")
	data.Set("hub.lease_seconds", "3")

	// Request itself
	req, err := http.NewRequest("POST", "http://localhost:4000/callback/"+callback, strings.NewReader(data.Encode()))
	if err != nil {
		panic(err)
	}

	// Headers
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Content-Length", strconv.Itoa(len(data.Encode())))

	httpmock.Deactivate()
	// Make the request
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		panic(err)
	}
	httpmock.Activate()

	// Should be a 200
	if resp.StatusCode != 200 {
		t.Fatalf("Status code is %d instead of 200", resp.StatusCode)
	}

	// Should have gotten the challenge parrotted back
	if respBody, err := ioutil.ReadAll(resp.Body); err == nil {
		if string(respBody) != "kitties" {
			t.Fatalf("Response is {%v} instead of {kitties}", respBody)
		}
	} else {
		t.Fatalf("Failed to parse body with err {%v}", err)
	}
}
