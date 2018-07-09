package subscriber

import (
	"bufio"
	"io"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/peterhellberg/link"
	"golang.org/x/net/html"
)

/*
	According to https://www.w3.org/TR/websub/#subscriber, a Subscriber
	is a service that discovers hubs and topics.
	According to https://www.w3.org/TR/websub/#conformance-classes, a Subscriber
	MUST:
	- support specific content-delivery mechanisms
	- send subscription requests according to the spec
	- acknowledge content-delivery requests with a HTTP 2xx code
	MAY:
	- request specific lease durations, related to the subscription
	- include a secret in the sub request.  If it does, then it
		- MUST use the secret to verify the signature in the content delivery request
	- request that a subscription be deactivated with an unsubscribe mechanism
*/

// Client subscribes to topic hubs, following the websub protocol
type Client struct {
	hostname        string
	topicsToHubs    map[string]map[string]struct{}
	topicURLsToSelf map[string]string
}

// NewClient creates and returns a new subscription client
func NewClient() *Client {
	return &Client{
		hostname:        "best",
		topicsToHubs:    make(map[string]map[string]struct{}),
		topicURLsToSelf: make(map[string]string),
	}
}

// GetHubsForTopic returns all hubs associated with a given topicURL
func (sc *Client) GetHubsForTopic(topicURL string) []string {
	hubs := make([]string, len(sc.topicsToHubs[topicURL]))
	if set, exists := sc.topicsToHubs[topicURL]; exists {
		for url := range set {
			hubs = append(hubs, url)
		}
	}
	return hubs
}

// DiscoverTopic is a request to a given topic url.
// Recipient: typically a publisher, but hubs implement it too.
// Response: a list of hubs who forward the content associated with this topic
func (sc *Client) DiscoverTopic(topicURL string) {
	// Form the request
	req, err := http.NewRequest("GET", topicURL, nil)
	if err != nil {
		panic(err)
	}
	// req.Host = sc.hostname

	// Make the request
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		panic(err)
	}

	log.Printf("Sent discovery request to %s\n", topicURL)

	// At this point, we have to choose from a few ways of extracting links
	// [1] From header (must check this always)
	// [2] From body in html, json, xml, etc (only check if header is missing links)

	// Allocate the mapping
	if _, ok := sc.topicsToHubs[topicURL]; !ok {
		sc.topicsToHubs[topicURL] = make(map[string]struct{})
	}

	log.Printf("Parsing discovery response from %s\n", topicURL)

	// Parse from the header
	hubURLs, selfURL := parseFromHeader(resp.Header)

	// We failed to find any header links
	if len(selfURL) == 0 {
		log.Printf("Failed to find links in header from %s\n", topicURL)
		// No links were in the header, have to check the body
		switch resp.Header.Get("Content-Type") {
		case "text/html; charset=UTF-8":
			hubURLs, selfURL := parseLinksFromHTML(resp.Body)
			sc.topicURLsToSelf[topicURL] = selfURL
			for hubURL := range hubURLs {
				sc.topicsToHubs[topicURL][hubURL] = struct{}{}
			}
		case "text/xml":
		}
	} else {
		log.Printf("Found links in the header of the response from %s\n", topicURL)
		log.Printf("Hubs found: {%v}", hubURLs)
		log.Printf("Self reported: {%v}", selfURL)
		sc.topicURLsToSelf[topicURL] = selfURL
		for hubURL := range hubURLs {
			sc.topicsToHubs[topicURL][hubURL] = struct{}{}
		}
	}
}

func (sc *Client) SubscribeToTopic(topicURL string) {
	for _, postURL := range sc.topicURLsToSelf {
		data := make(url.Values)
		data.Set("hub.callback", "http://172.104.24.141:9090/callback")
		data.Set("hub.mode", "subscribe")
		data.Set("hub.topic", postURL)

		log.Printf("Pinging %v\n", postURL)

		req, _ := http.NewRequest("POST", postURL, strings.NewReader(data.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.Header.Set("Content-Length", strconv.Itoa(len(data.Encode())))

		// Make the request
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			panic(err)
		}

		s, _ := bufio.NewReader(resp.Body).ReadString(byte('\t'))

		log.Printf("Response received with resp {%s}, header {%v}, code {%v}", s, resp.Header, resp.StatusCode)
	}
}

// Parse links from the header of an http reply
// Will return an un-init'd map and an empty string if no link headers exist
func parseFromHeader(header http.Header) (hubURLs map[string]struct{}, selfURL string) {
	hubURLs = make(map[string]struct{})
	group := link.ParseHeader(header)

	log.Printf("Recvd header {%v}", header)

	for _, link := range group {
		switch link.Rel {
		case "self":
			log.Printf("Found self link {%v}", link.URI)
			selfURL = link.URI
		case "hub":
			log.Printf("Found hub link {%v}", link.URI)
			hubURLs[link.URI] = struct{}{}
		}
	}

	return hubURLs, selfURL
}

// Parse links from the body of an http reply, assumes that the body is in html
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

	log.Printf("Found: {%v} {%v}", hubURLs, selfURL)

	return hubURLs, selfURL
}
