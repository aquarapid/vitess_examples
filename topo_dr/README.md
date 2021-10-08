# Topo Disaster Recovery

If a topo disaster has wiped out your global topo
(see https://vitess.io/docs/reference/features/topology-service/#recovery)
the recovery options are:

  * Restore the topo from a backup (the backup and recovery procedures
    here would be specific to your topology server implementation).  This
    is the preferred procedure.
  * Alternatively, you will have to rebuild the topo server info.  This will
    cause some downtime, since you need to restart many components.  However,
    you should be able to run for a short amount of time without the global
    topo, assuming you run into no other problems (e.g. need the topo to do
    a Reparent, etc.)


Steps to rebuild the global topo are roughly as follows:

  * Bring up a new (empty) topo server (etcd, zk, consul)
  * Reinitialize your CellInfo for each cell (we assume a single topo
    server here doing both global and local topo duties). Either way,
    you should follow the same procedure you did when you initially
    bootstrapped your Vitess topology server.  Here we have a single cell
    called `zone1`:
  ```
$ vtctl -topo_implementation etcd2 -topo_global_server_address 192.168.0.212:2379 -topo_global_root /vitess/global AddCellInfo -root /vitess/zone1 -server_address 192.168.0.212:2379 zone1
  ```
  * Now, assuming that your topo address is still the same, running
    instances of `vtctld` should be functional again, and you should be able
    to use `vtctlclient`, e.g.:
  ```
  $ vtctlclient -server 192.168.0.212:15999  GetCellInfoNames
  zone1
  ```
  * Now, restart all the tablets, they will recreate topo info.
  * After this, your tablet records should be back, e.g.:
  ```
$ vtctlclient -server 192.168.0.212:15999 ListAllTablets
zone1-0000000100 keyspace1 0 replica 192.168.0.212:15100 192.168.0.212:17100 [] <null>
zone1-0000000101 keyspace1 0 replica 192.168.0.212:15101 192.168.0.212:17101 [] <null>
zone1-0000000102 keyspace1 0 replica 192.168.0.212:15102 192.168.0.212:17102 [] <null>
  ```
  * Re-apply VSchema, if necessary (it would have been lost with the topo), e.g.:
  ```
$ vtctlclient -server 192.168.0.212:15999 ApplyVSchema -vschema_file `pwd`/vschema.json keyspace1
  ```
  * Re-add any routing rules with `vtctlclient ApplyRoutingRules` (if necessary)
  * Restart vtgates; to clear out old healthcheck info (potentially optional)
  * Run TER on each keyspace/shard combo to let Vitess now which of
    the tablets were the primaries before you restarted, e.g.:
  ```
$ vtctlclient -server 192.168.0.212:15999 TabletExternallyReparented zone1-0000000100
  ```
  * Repeat as necessary for additional cells and keyspaces.
  * Now, you should have a functioning cluster and keyspace(s), e.g.:
  ```
$ vtctlclient -server 192.168.0.212:15999 ListAllTablets
zone1-0000000100 keyspace1 0 primary 192.168.0.212:15100 192.168.0.212:17100 [] 2021-10-08T03:50:41Z
zone1-0000000101 keyspace1 0 replica 192.168.0.212:15101 192.168.0.212:17101 [] <null>
zone1-0000000102 keyspace1 0 replica 192.168.0.212:15102 192.168.0.212:17102 [] <null>
  ```

