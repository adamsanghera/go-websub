# Storage

This package defines an interface (and a sqlite3 implementation thereof), which acts as a centralized source of truth regarding the state of subscriptions (active, inactive, etc) managed by a Subscriber.

## Lifecycle of a Susbcription

Extrapolating from the W3 recommendation, there are many states a subscription might find itself in.  

It is important to keep track of these states, because they determine both the behavior of the Subscriber (i.e. how it handles and responds to both hub and client requests), and the behavior of the Hub, which is managing subscriptions across many different Subscribers.

The lifecycle looks something like this:

```none

      |---<---<---ending---<------<----|-------------------|
      |                                |                   |
[ack/denied]                        [unsub]             [unsub]
      |                                |                   |
    dead -[created]-> born -[ack]-> active -[renewal]-> renewing
      |                |              | |                  |
      |-<--[denied]-<--|-<-[denied]-<-| |--<--[ack]---<----|
      |                                                    |
      |------<----------<-----<---[denied]--<------<-------|

```

Note that a callback_url is created/valid at birth, and deleted/no-longer-valid at death.

## Goals of this package

The goal of this package is to enforces the above state machine vision of a subscription in a threadsafe way, using SQLite3 transactions.

A secondary goal is to accomplish this task efficiently in terms of CPU cycles and disk space.

## TODOs

1. Contemplate tests for uniquness of callbacks --> do we ever want to clean up / reclaim old callbacks?  Probably.  When?

1. Write more tests for:
   1. active/inactive (test the edge cases of paging)
1. Implement persistence
   1. active persistence (i.e. periodically flushing to disk during life)
   1. shutdown persistence (supporting flushing to disk on shutdown)
