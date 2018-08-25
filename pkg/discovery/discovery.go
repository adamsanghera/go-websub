package discovery

import (
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/peterhellberg/link"
	"golang.org/x/net/html"
)

// DiscoverTopic is a request to a given topic url.
//
// Recipient: typically a publisher, but hubs implement it too.
// Response: a list of hubs who forward the content associated with this topic
func DiscoverTopic(topic string) (map[string]struct{}, string, error) {
	// Form the request
	req, err := http.NewRequest("GET", topic, nil)
	if err != nil {
		return make(map[string]struct{}), "", err
	}

	// Make the request
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return make(map[string]struct{}), "", err
	}

	contentType := resp.Header.Get("Content-Type")

	// If header contains links, try to get them there.
	if _, ok := resp.Header["Link"]; ok {
		hubs, self := parseFromHeader(resp.Header)
		if self != "" {
			return hubs, self, nil
		}
	}

	// If the goods weren't in the header, go deeper
	if strings.Contains(contentType, "text/html") {
		return parseLinksFromHTML(resp.Body)
	} else if strings.Contains(contentType, "text/xml") {
		// TODO(adam) handle xml [atom, rss, etc.]
	}

	return make(map[string]struct{}), "", errors.New("Response from URL provided was not parseable")
}

// Parse links from the header of an http reply
// Will return an un-init'd map and an empty string if no link headers exist
func parseFromHeader(header http.Header) (map[string]struct{}, string) {
	hubURLs := make(map[string]struct{})
	selfURL := ""
	group := link.ParseHeader(header)

	for _, link := range group {
		switch link.Rel {
		case "self":
			selfURL = link.URI
		case "hub":
			hubURLs[link.URI] = struct{}{}
		}
	}

	return hubURLs, selfURL
}

// Parse links from the body of an http reply, assumes that the body is in html
// TODO(adam) make this code much more legible
func parseLinksFromHTML(htmlReader io.Reader) (map[string]struct{}, string, error) {
	tokenizer := html.NewTokenizer(htmlReader)

	hubURLs := make(map[string]struct{})
	selfURL := ""
	inHead := false
	parsing := true
	for parsing {
		tt := tokenizer.Next()
		switch tt {

		// We're looking for links embedded in heads
		case html.StartTagToken:
			t := tokenizer.Token()
			if t.Data == "head" {
				inHead = true
			} else if t.Data == "link" && inHead {
				// We've found a link tag, now we need to validate that it has the following components:
				// 1. rel, which is one of (a) hub or (b) self
				// 2. href, which is a valid url
				var href string
				isHub := false
				isSelf := false
				for _, a := range t.Attr {
					if a.Key == "rel" {
						isHub = a.Val == "hub"
						isSelf = a.Val == "self"
						if !(isHub || isSelf) {
							break
						}
					} else if a.Key == "href" {
						href = a.Val
					}
				}
				if isHub || isSelf && len(href) != 0 {
					if isHub {
						hubURLs[href] = struct{}{}
					} else {
						selfURL = href
					}
				}

			}
		// Stop parsing once we exit the head
		case html.EndTagToken:
			tn, _ := tokenizer.TagName()
			if len(tn) == 4 {
				if string(tn) == "html" {

					parsing = false
					break
				}
				if string(tn) == "head" {
					inHead = false
					parsing = false
					break
				}
			}
		// Obviously, stop parsing if we hit an error token
		case html.ErrorToken:
			parsing = false
			return make(map[string]struct{}), "", errors.New("Received malformed html from target")
		}
	}

	if len(selfURL) == 0 {
		return hubURLs, selfURL, errors.New("Target did not provide a self reference")
	} else {
		return hubURLs, selfURL, nil
	}
}
