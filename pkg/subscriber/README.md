# Subscriber

[![GoDoc](https://godoc.org/github.com/adamsanghera/go-websub/cmd/subscriber?status.svg)](https://godoc.org/github.com/adamsanghera/go-websub/cmd/subscriber)

## Description / Spec

Subscriber is a Go server library that implements the [W3 Group's WebSub protocol](https://www.w3.org/TR/websub/), a broker-supported pub-sub architecture built on top of HTTP.

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

## Implementation

This package implements the subscriber using a few layers of abstraction:

The `subscriber` object, defined in `subscriber.go` is the open door to this package's functionality.  It is composed of a few logical parts:

1. The `storage` object, defined in the included `storage` package, is used to update, query, and maintain all state related to subscriptions
1. A dynamic collection of channels, used to communicate with background routines that own their respective subscriptions
   - When a subscription is created, it is added to storage as launched, and a maintainer routine is spawned to manage the subscription.
   - When a client or hub command is received by the `subscriber` object regarding the callback url, it is passed off to the maintenance routine via a callback_url-indexed channel.
   - The maintainer will shut down when the subscription ends, due to either a client or hub-triggered shutdown.
1. A longrunning `net/http` server, which is used to support callbacks

### Life cycle of subscriber object

The `subscriber` object has three stages in its life cycle:

1. Birth
   - `storage` initialized
   - `net/http` server initialized
2. Normal state
   - Processes commands from the client application(s)
   - Listens for responses from hubs, and processes them accordingly
3. Shutdown
   - Sends a shutdown signal to the client's callback server
   - (optional) Flushes SQLite3 database of discovered hubs to file.

### Important Assumptions In the Implementation

- Sticky subscriptions (i.e. auto-renewing subscriptions) are the only subscriptions we want
- When the Subscriber service dies, all of its subscriptions are cancelled (i.e. no hot-startups)
- For every (topic <--> hub) tuple, there will be only 1 active subscription maintained.

### TODO's

[ ] Handle callbacks, leveraging the storage package

### Whims

- Think about making it an option to persist subscriptions "after death"
- Think about making sticky-subscriptions optional.
