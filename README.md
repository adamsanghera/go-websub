# A WIP Implementation of WebSub

[![Go Report Card](https://goreportcard.com/badge/github.com/adamsanghera/go-websub?style=flat)](https://goreportcard.com/report/github.com/adamsanghera/go-websub)
[![Travis CI](https://travis-ci.com/adamsanghera/go-websub.svg?branch=master)](https://travis-ci.com)
[![codecov](https://codecov.io/gh/adamsanghera/go-websub/branch/master/graph/badge.svg)](https://codecov.io/gh/adamsanghera/go-websub)

Doing this for fun.  Watch me go!

## Design notes

- Active vs Inactive state is implicitly defined, using a timestamp-expiration mechanism (inspired by how Chubby handles leases)
  - Inactive: expiration prior to `now()`
  - Active: expiration in the future
- Abrupt cancels (i.e. in cases of denial, or a user-initiated cancel) are a two-phase process
  1. The subscription server kills the renewal routine
  1. SQLite Update request, setting expiration time to `now()`
  - In the worst case of arbitrary failure, the client believes its lease active, until either (1) the existing timestamp expires, or (2) a subsequent renewal is rejected
- Successful renewal routines update the expiration time upon ACK
