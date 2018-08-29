# Subscriber

[![GoDoc](https://godoc.org/github.com/adamsanghera/go-websub/cmd/subscriber?status.svg)](https://godoc.org/github.com/adamsanghera/go-websub/cmd/subscriber)

## Description / Spec

package subscribe is a Go server library that implements the [W3 Group's WebSub protocol](https://www.w3.org/TR/websub/), a broker-supported pub-sub architecture built on top of HTTP.

According to [the spec](https://www.w3.org/TR/websub/#subscriber) a Subscriber is a service that discovers hubs, and subscribes to topics.  More specifically, as described [here](https://www.w3.org/TR/websub/#conformance-classes), a Subscriber must conform to the following specs:

MUST:

- support specific content-delivery mechanisms
- send subscription requests according to the spec
- acknowledge content-delivery requests with a HTTP 2xx code

MAY:

- request specific lease durations, related to the subscription
- include a secret in the sub request.  If it does, then it
  - MUST use the secret to verify the signature in the content delivery request
- request that a subscription be deactivated with an unsubscribe mechanism

This package implements the above requirements with the server struct.

## Lifecycle

The server has three stages in its life cycle.

1. Birth
   - All data structures are initialized
   - An http server is created, to support [callbacks](https://www.w3.org/TR/websub/#hub-verifies-intent)
   - The callback endpoint is registered
2. Normal state
   - Processes subscription/unsubscription/discovery commands in parallel
   - Should never panic, only log errors
3. Shutdown
   - Sends a shutdown signal to the client's callback server
   - Prunes active subscriptions, sending cancel signals to all sleeping routines [ should have some way of short-circuiting outgoing subscription calls, too, so that those privileged few that escape this cancel barrage don't go on living ]

## Assumptions

- Cient is a long-running service
- Sticky subscriptions (i.e. auto-renewing subscriptions) are the only subscriptions we want

## TODO's

[x] Tests/Benchmarks that throttle parallelism (am pretty confident that this will work, but that efficiency can be improved)
[ ] Refactor to use channels > mutexes

## Whims

- Think about how we can leverage channels instead of mutexes.

## Copy-pasta for implementation report

Programming Language(s): Go

Developer(s): [Adam Sanghera](https://github.com/adamsanghera)

Answers are:

- [x] Confirmed via websub.rocks (for applicable results)
- [ ] All results are self-reported

### Discovery

- [x] 100: HTTP header discovery - discovers the hub and self URLs from HTTP headers
- [ ] 101: HTML tag discovery - discovers the hub and self URLs from the HTML `<link>` tags
- [ ] 102: Atom feed discovery - discovers the hub and self URLs from the XML `<link>` tags
- [ ] 103: RSS feed discovery - discovers the hub and self URLs from the XML `<atom:link>` tags
- [ ] 104: Discovery priority - prioritizes the hub and self in HTTP headers over the links in the body

### Subscription

- [ ] 1xx: Successfully creates a subscription
- [ ] 200: Subscribing to a URL that reports a different rel=self
- [ ] 201: Subscribing to a topic URL that sends an HTTP 302 temporary redirect
- [ ] 202: Subscribing to a topic URL that sends an HTTP 301 permanent redirect
- [ ] 203: Subscribing to a hub that sends a 302 temporary redirect
- [ ] 204: Subscribing to a hub that sends a 301 permanent redirect
- [ ] 205: Rejects a verification request with an invalid topic URL
- [ ] 1xx: Requests a subscription using a secret (optional)
  - Please select the signature method(s) that the subscriber recognizes. All methods listed below are currently acceptable for the hub to choose:
  - [ ] sha1
  - [ ] sha256
  - [ ] sha384
  - [ ] sha512
- [ ] 1xx: Requests a subscription with a specific `lease_seconds` (optional, hub may ignore)
- [ ] Callback URL is unique per subscription (should)
- [ ] Callback URL is an unguessable URL (should)
- [ ] 1xx: Sends an unsubscription request