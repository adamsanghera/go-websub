# A WIP Implementation of WebSub

[![Go Report Card](https://goreportcard.com/badge/github.com/adamsanghera/go-websub?style=flat)](https://goreportcard.com/report/github.com/adamsanghera/go-websub)
[![Travis CI](https://travis-ci.com/adamsanghera/go-websub.svg?branch=master)](https://travis-ci.com)

Doing this for fun.  Watch me go!

## Subscriber

Programming Language(s): Go

Developer(s): [Adam Sanghera](https://github.com/adamsanghera)

Answers are:

* [x] Confirmed via websub.rocks (for applicable results)
* [ ] All results are self-reported

## Discovery

* [x] 100: HTTP header discovery - discovers the hub and self URLs from HTTP headers
* [ ] 101: HTML tag discovery - discovers the hub and self URLs from the HTML `<link>` tags
* [ ] 102: Atom feed discovery - discovers the hub and self URLs from the XML `<link>` tags
* [ ] 103: RSS feed discovery - discovers the hub and self URLs from the XML `<atom:link>` tags
* [ ] 104: Discovery priority - prioritizes the hub and self in HTTP headers over the links in the body

## Subscription

* [ ] 1xx: Successfully creates a subscription
* [ ] 200: Subscribing to a URL that reports a different rel=self
* [ ] 201: Subscribing to a topic URL that sends an HTTP 302 temporary redirect
* [ ] 202: Subscribing to a topic URL that sends an HTTP 301 permanent redirect
* [ ] 203: Subscribing to a hub that sends a 302 temporary redirect
* [ ] 204: Subscribing to a hub that sends a 301 permanent redirect
* [ ] 205: Rejects a verification request with an invalid topic URL
* [ ] 1xx: Requests a subscription using a secret (optional)
  * Please select the signature method(s) that the subscriber recognizes. All methods listed below are currently acceptable for the hub to choose:
  * [ ] sha1
  * [ ] sha256
  * [ ] sha384
  * [ ] sha512
* [ ] 1xx: Requests a subscription with a specific `lease_seconds` (optional, hub may ignore)
* [ ] Callback URL is unique per subscription (should)
* [ ] Callback URL is an unguessable URL (should)
* [ ] 1xx: Sends an unsubscription request

## Distribution

* [ ] 300: Returns HTTP 2xx when the notification payload is delivered
* [ ] 1xx: Verifies a valid signature for authenticated distribution
* [ ] 301: Rejects a distribution request with an invalid signature
* [ ] 302: Rejects a distribution request with no signature when the subscription was made with a secret

(1xx denotes that you can can use any of the 100-104 tests to confirm this feature)

## Publisher

## Hub