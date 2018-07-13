package common

import (
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
func DiscoverTopic(topic string) (map[string]struct{}, string) {
	// Form the request
	req, err := http.NewRequest("GET", topic, nil)
	if err != nil {
		panic(err)
	}

	// Make the request
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		panic(err)
	}

	contentType := resp.Header.Get("Content-Type")

	// If header contains links, try to get them there.
	if _, ok := resp.Header["Link"]; ok {
		hubs, self := parseFromHeader(resp.Header)
		if self != "" {
			return hubs, self
		}
	}

	// If the goods weren't in the header, go deeper
	if strings.Contains(contentType, "text/html") {
		return parseLinksFromHTML(resp.Body)
	} else if strings.Contains(contentType, "text/xml") {
		// muh xml
	}

	return make(map[string]struct{}), ""
}

// Parse links from the header of an http reply
// Will return an un-init'd map and an empty string if no link headers exist
func parseFromHeader(header http.Header) (hubURLs map[string]struct{}, selfURL string) {
	hubURLs = make(map[string]struct{})
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
func parseLinksFromHTML(htmlReader io.Reader) (hubURLs map[string]struct{}, selfURL string) {
	tokenizer := html.NewTokenizer(htmlReader)

	hubURLs = make(map[string]struct{})

	inHead := false
	parsing := true
	for parsing {
		tt := tokenizer.Next()
		switch tt {

		// We're looking for links embedded in heads
		case html.StartTagToken:
			tn, _ := tokenizer.TagName()
			if len(tn) == 4 {
				if string(tn) == "head" {
					inHead = true
				} else if inHead && string(tn) == "link" {
					relFound := false
					hrefFound := false
					isHub := true
					href := ""

					k, val, more := tokenizer.TagAttr()

					switch string(k) {
					case "rel":
						isHub = string(val) == "hub"
					case "href":
						href = string(val)
					}

					relFound = relFound || string(k) == "rel"
					hrefFound = hrefFound || string(k) == "href"

					for more {
						k, val, more = tokenizer.TagAttr()
						switch string(k) {
						case "rel":
							isHub = string(val) == "hub"
						case "href":
							href = string(val)
						}
					}

					if href != "" {
						if isHub {
							hubURLs[href] = struct{}{}
						} else {
							selfURL = href
						}
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
			break
		}
	}

	return hubURLs, selfURL
}
