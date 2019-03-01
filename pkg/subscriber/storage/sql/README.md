# SQL Implementation

## Idempotency is king

The state machine is much simpler when it comes strings-unnattached, aka without side effects aka idempotent.  There's much less error-handling situations for the caller to have to deal with, when everything is an idempotent, atomic update that binary fails or succeeds.

To achieve this, I'm going to make indexing links, and adding callbacks to links, part of the same table.  This way, when we're adding a callback, it's an update operation touching exactly one row (indexed by [topic, hub]).

This means the primary key is now [topic, hub].  I'm ok with this, because I'm ok with only permitting 1 subscription per user.  The big benefit to this refactor, is that NewCallback becomes an update, rather than an insert or delete.  This will reap benefits like those reaped by making `invalidate` and `extendLease` into idempotent update queries.  Exciting.  This will make client code even simpler :)

