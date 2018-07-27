# Subscriber

[![GoDoc](https://godoc.org/github.com/adamsanghera/go-websub/cmd/subscriber?status.svg)](https://godoc.org/github.com/adamsanghera/go-websub/cmd/subscriber)

## Description / Spec

Package subscriber is a Go client library that implements the [W3 Group's WebSub protocol](https://www.w3.org/TR/websub/), a broker-supported pub-sub architecture built on top of HTTP.

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

This package implements the above requirements with the Client struct.

## Lifecycle

The client has three stages in its life cycle.

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

- Tests/Benchmarks that throttle parallelism (am pretty confident that this will work, but that efficiency can be improved)

## Whims

- Think about how we can leverage channels instead of mutexes.