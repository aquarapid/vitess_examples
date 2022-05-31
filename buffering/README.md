# VTGate buffering

VTGate in Vitess supports the **buffering** of queries in certain situations.
The original intention of this feature was to **reduce** (but not necessarily
eliminate) downtime during planned failovers (a.k.a. PRS -
`PlannedReparentShard` operations).  It has been extended to provide buffering
in some additional (planned) failover situations, e.g. during resharding.

Note that buffering is not intended for, or active during, unplanned failovers
or other unplanned issues with a `PRIMARY` tablet during normal operations.

As you may imagine if you think about the problem, buffering can be
somewhat involved, and there are a number of tricky edge cases. We will
discuss this in the context of an application's experience, starting with
the simplest case, that of buffering during a PRS (`PlannedReparentShard`)
operation.

## VTGate parameters to enable buffering

First, let us cover the parameters that need to be set in VTGate to enable
buffering:
  * `-enable_buffer`:  Enables buffering.  **Not enabled by default**
  * `-enable_buffer_dry_run`:  Enable logging of if/when buffering would
  trigger, without actually buffering anything. Useful for testing.
  Default: `false`
  * `-buffer_implementation`:  Default: `healthcheck`.  If you want buffering
  during resharding, you need to use `keyspace_events` here.
  * `-buffer_size`:  Default: `10`.  The maximum **number** of in-flight
  buffered requests. This default is obviously way too small for anything but
  testing, and should be adjusted.
  * `-buffer_drain_concurrency`:  Default: `1`.  If the buffer is of any
  significant size, you probably want to increase this proportionally.
  * `-buffer_keyspace_shards`:  Can be used to limit buffering to only
  certain keyspaces. Should not be necessary in most cases.
  * `-buffer_max_failover_duration`:  Default: `20s`.  If buffering is active
  for longer than this (from when the first request was buffered: TODO check), 
  stop buffering and return errors to the buffered requests.
  * `-buffer_window`: Default: `10s`.  The maximum time any individual request
  should be buffered for. Should probably be less than the value for
  `-buffer_max_failover_duration`. Adjust according to your application
  requirements.
  * `-buffer_min_time_between_failovers`: Default `1m`. If consecutive
  failovers for a shard happens within less than this duration, do **not**
  buffer again. This avoids "endless" buffering if there are consecutive
  failovers, and makes sure that the application will eventually receive
  errors that will allow it (or the application client) to take appropriate
  action within a bounded amount of time.

## Types of queries that can be buffered

 * Only requests to tablets of type `PRIMARY` are buffered. In-flight requests
 to a `REPLICA` in the process of transitioning to `PRIMARY` because of a PRS
 should be unaffected, and do not require buffering.

## What happens during buffering

Fundamentally we are:
 * Holding up and buffering queries to the `PRIMARY` tablet for a shard
 * ... in the hope that the failover to a replica happens quickly enough
 * ... that we can then drain the buffered queries to the new `PRIMARY`
 tablet.
 * Allowing the client to experience a "pause" in requests rather than
 lots of errors.

Note that this process is not guaranteed to eliminate errors to the
application, but rather reduce them or make them less frequent. The application
should still endeavor to handle errors appropriately if/when they
occur (e.g. unplanned outages/failovers, etc.)

## How does it work?

Simplifying considerably:
  * All buffering is done in `vtgate`
  * When a shard begins a failover or resharding event, and a query is sent
  from `vtgate` to `vttablet`, `vttablet` will return a certain type of error
  to `vtgate` (`vtrpcpb.Code_CLUSTER_EVENT`).
  * This error indicates to `vtgate` that it is appropriate to buffer this
  request.
  * Separately the various timers associated with the flags above are being
  maintained to timeout and return errors to the application when appropriate,
  e.g. if an individual request was buffered for too long;  or if buffering
  start "too long" ago.
  * When the failover is complete, and the tablet starts accepting queries
  again, we start draining the buffered queries, with a concurrency as
  indicated by the `-buffer_drain_concurrency` parameter.
  * When the buffer is drained, the buffering is complete.  We maintain a
  timer based on `-buffer_min_time_between_failovers` to make sure we
  do not buffer again if another failover starts within that period.


## What the application sees

TODO:  Error X if Y:
  * max duration exceeded
  * etc.
