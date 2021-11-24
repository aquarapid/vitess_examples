## How to use `topo2topo` to migrate a topology server

Sometimes a Vitess deployment may start with one topology server implementation,
and build a Vitess infrastructure around it, only to later decide to switch
topology server implementations.

There is a CLI tool provided with Vitess to assist in a migration like
this, called `topo2topo`.  Here is an example of how to migrate a (very)
simple Vitess installation from a Vitess topology implementation of using
`consul`, to one using `etcd`.  You should be able to generalize this
to more complex installations.

This walkthrough assumes that your `consul` server is up and running at
`x.x.x.x:8500` and all your current Vitess components are pointing to it.
This generalizes for real-world cases where you will have multiple `consul`
IP/port endpoints.  We also assume a situation where the global and local
topo services are shared both on the source topology service, and for the
target topology service.

We will also assume we have only one cell, called `zone1`, and two keyspaces,
`sharded` and `unsharded`.  Lastly, we will assume the topology global root
is at `/vitess/global` and the `zone1` root at `/vitess/zone1`.

For the duration of the migration, we assume a stable topology, with no
resharding or reparenting in flight.  If any reparents need to be performed
during the migration process, some steps will have to be re-run, at the very
least.


## Steps:

 * Configure the new topology service (`etcd`) in this case, and make sure it
 is running.  We assume the endpoint address for the `etcd` topology service
 is `y.y.y.y:2379`
 * Run `vtctl AddCellInfo` to initialize each of the cells you are migrating
 on the target topology service, e.g. with our single cell called `zone1`,
 and assuming that the global and local topology services share the same
 `etcd` cluster:
 ```sh
 $ vtctl -topo_implementation etcd2 -topo_global_server_address y.y.y.y:2379 -topo_global_root /vitess/global AddCellInfo -root /vitess/zone1 -server_address y.y.y.y:2379 zone1
 ```
 * Run `topo2topo` with the right source and target topology flags:
   * `-from_implementation`, `-from_root`, `-from_server`
   * `-to_implementation`, `-to_root`, `-to_server`
   * You will have to run it multiple times, to copy each of the different
   sets of objects we are interested in, e.g.:
   ```sh
   $ topo2topo -from_implementation consul -from_root vitess/global -from_server x.x.x.x:8500 -to_implementation etcd2 -to_root /vitess/global -to_server y.y.y.y:2379 -do-keyspaces
   $ topo2topo -from_implementation consul -from_root vitess/global -from_server x.x.x.x:8500 -to_implementation etcd2 -to_root /vitess/global -to_server y.y.y.y:2379 -do-shards
   $ topo2topo -from_implementation consul -from_root vitess/global -from_server x.x.x.x:8500 -to_implementation etcd2 -to_root /vitess/global -to_server y.y.y.y:2379 -do-routing-rules
   $ topo2topo -from_implementation consul -from_root vitess/global -from_server x.x.x.x:8500 -to_implementation etcd2 -to_root /vitess/global -to_server y.y.y.y:2379 -do-shard-replications
   $ topo2topo -from_implementation consul -from_root vitess/global -from_server x.x.x.x:8500 -to_implementation etcd2 -to_root /vitess/global -to_server y.y.y.y:2379 -do-shard-tablets
   ```
 * Note that unless there is an error, `topo2topo` will not produce output.
 * You can validate that the various objects have been copied by dumping the
   keys in the `etcd` server:
   ```sh
   $  ETCDCTL_API=3 etcdctl --endpoints="http://y.y.y.y:2379" get "" --prefix --keys-only
   <output omitted>
   ```
 * Now, run `vtctl RebuildKeyspaceGraph` for each keyspace, using the **new**
   topology flags:
   ```sh
   $ vtctl  -topo_implementation etcd2 -topo_global_server_address y.y.y.y:2379 -topo_global_root /vitess/global RebuildKeyspaceGraph unsharded
   $ vtctl  -topo_implementation etcd2 -topo_global_server_address y.y.y.y:2379 -topo_global_root /vitess/global RebuildKeyspaceGraph sharded
   ```
 * Now, run `vtctl RebuildVSchemaGraph` using the **new** topology flags:
   ```sh
   $ vtctl  -topo_implementation etcd2 -topo_global_server_address y.y.y.y:2379 -topo_global_root /vitess/global RebuildVSchemaGraph
   ```
 * At this point, the new topology should be fully built out.  You can use
   `etcdctl` if you want to do further manual data validation on the target
   topo.
 * Now, you can start reconfiguring Vitess components to point to the 
   new topology service, and restarting them:
   * The flags you will have to change (in the simple case) are, as used with
     the `vtctl` example command above:
     * `-topo_implementation consul` becomes `-topo_implementation etcd2`
     * `-topo_global_server_address x.x.x.x:8500` becomes `-topo_global_server_address y.y.y.y:2379`
     * `-topo_global_root vitess/global` becomes `-topo_global_root /vitess/global`
   * First, reconfigure and restart all `vtgate` processes
     * Validate in turn that they can still see the vttablets via
     `show vitess_tablets` after the restart.
   * Next, reconfigure and restart all `vttablet` processes
   * Last, reconfigure and restart all `vtctld` processes
     * Validate that everything looks as expected via the `vtctld` topology
     browser
 * Validate that your old topo server is not seeing any connections
   from Vitess components anymore.
 * You should now be able to decomission your source topology service.

## Using `topo2topo -compare`

You can use the `-compare` flag for `topo2topo` to validate objects between
the source (`-from_`) and target (`-to_`) topology servers. However, note
the usage:  `-compare` only makes sense in the context of one of the `do-*`
flags, i.e.:
  * `-do-keyspaces`
  * `-do-shards`
  * `-do-routing-rules`
  * `-do-shard-replications`
  * `-do-tablets`

You will have to run `-compare` with each of these in turn to validate that
the data in the source and target topology servers are indeed the same.

As an partial example:

```sh
$ topo2topo -from_implementation consul -from_root vitess/global -from_server x.x.x.x:8500 -to_implementation etcd2 -to_root /vitess/global -to_server y.y.y.y:2379 -do-shards -compare
Topologies are in sync
```

However, if we now do a failover (`PlannedReparentShard`) on the `unsharded`
keyspace of our Vitess cluster, this will update the topology info only
in the target topology.  After this, if we run the compare again:

```sh
$ topo2topo -from_implementation consul -from_root vitess/global -from_server x.x.x.x:8500 -to_implementation etcd2 -to_root /vitess/global -to_server y.y.y.y:2379 -do-shards -compare
F1124 07:52:33.327520  109957 topo2topo.go:108] Compare shards failed: Code: FAILED_PRECONDITION
Shard 0 for keyspace: unsharded does not match between from and to topology
```

which makes sense, since the global shard record for shard `/0` of the
`unsharded` keyspace was updated with the new primary's information when
the `PlannedReparentShard` was performed.

