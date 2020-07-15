We are going to demonstrate an example of a vStream client 
(`vstream_test_client.go`), by using the Vitess local examples:

  - Checkout the Vitess code (https://github.com/vitessio/vitess.git) for 
    a local build.
  - Build Vitess, make sure no Vitess components are running, and start 
the parts of the local example, i.e. run:

```
$ source dev.env
$ make clean ; make

(Next, ONLY if necessary to clean up from a previous examples run)
$ pkill -9 -f -e vtdataroot
$ rm -rf ~/go/vtdataroot/*

$ cd examples/local/
$ for i in `ls -1 [1-2]*.sh` ; do echo $i ; ./$i ; sleep 5 ; done ; ./301_customer_sharded.sh
```

  - Now you have an unsharded `customer` keyspace setup.
    Let us start our example vStream client against this, and then shard it.
  - Run (in a separate terminal), from this repo:

```
$ go run vstream_test_client.go -vtgate=localhost:15991 -keyspace=customer -tablet_type=replica -pos='[ {"shard":"", "gtid":"current"} ]'
vtgate connecting to: localhost:15991
vgtid: shard_gtids:<keyspace:"customer" gtid:"current" >
tablet_type: REPLICA

```

  - This should print nothing further (yet)
  - In a different terminal, connect to use `customer` database and insert some data:

```
$ mysql -A -u root -P 15306 -h 127.0.0.1 customer
.
.
mysql> insert into corder (order_id, customer_id, sku, price) values (23, 107, 123, "124445");
Query OK, 1 row affected (0.01 sec)

mysql> insert into customer (customer_id, email) values (9, "123");
Query OK, 1 row affected (0.01 sec)
```

  - You should now see something like this in the vStream client terminal (leave it running):

```
Event log timestamp: 1593314922 --> 2020-06-28 03:28:42 +0000 UTC
+-----------------+--------+----------+-------------+-----+--------+
|      table      |   op   | order_id | customer_id | sku | price  |
+-----------------+--------+----------+-------------+-----+--------+
| customer.corder | INSERT |       23 |         107 | 123 | 124445 |
+-----------------+--------+----------+-------------+-----+--------+
VGTID after event:  shard_gtids:<keyspace:"customer" shard:"0" gtid:"MySQL56/d714417e-b8ee-11ea-ad61-001e677affd5:1-36" >

Event log timestamp: 1593314931 --> 2020-06-28 03:28:51 +0000 UTC
+-------------------+--------+-------------+-------+
|       table       |   op   | customer_id | email |
+-------------------+--------+-------------+-------+
| customer.customer | INSERT |           9 |   123 |
+-------------------+--------+-------------+-------+
VGTID after event:  shard_gtids:<keyspace:"customer" shard:"0" gtid:"MySQL56/d714417e-b8ee-11ea-ad61-001e677affd5:1-37" >
```

  - Now, let's complete the sharding of the customer keyspace by running in the `examples/local` directory:

```
$ for i in `ls -1 30[2-5]*.sh` ; do echo $i ; ./$i ; sleep 5 ; done
```

  - This should now reshard the unsharded `customer` keyspace (shard `0`) into 2 shards:
    - `-80` and
    - `80-`
  - During the resharding process you will see "empty" vStream transactions 
    in the vStream client output, like:

```
Event log timestamp: 1593315092 --> 2020-06-28 03:31:32 +0000 UTC
VGTID after event:  shard_gtids:<keyspace:"customer" shard:"0" gtid:"MySQL56/d714417e-b8ee-11ea-ad61-001e677affd5:1-38" > 

Event log timestamp: 1593315092 --> 2020-06-28 03:31:32 +0000 UTC
VGTID after event:  shard_gtids:<keyspace:"customer" shard:"0" gtid:"MySQL56/d714417e-b8ee-11ea-ad61-001e677affd5:1-39" > 

Event log timestamp: 1593315092 --> 2020-06-28 03:31:32 +0000 UTC
VGTID after event:  shard_gtids:<keyspace:"customer" shard:"-80" gtid:"MySQL56/9e9047d4-b8ef-11ea-80a2-001e677affd5:1-27" > shard_gtids:<keyspace:"customer" shard:"80-" gtid:"MySQL56/b8b9ba74-b8ef-11ea-8ce3-001e677affd5:1-26" > 

Event log timestamp: 1593315092 --> 2020-06-28 03:31:32 +0000 UTC
VGTID after event:  shard_gtids:<keyspace:"customer" shard:"-80" gtid:"MySQL56/9e9047d4-b8ef-11ea-80a2-001e677affd5:1-28" > shard_gtids:<keyspace:"customer" shard:"80-" gtid:"MySQL56/b8b9ba74-b8ef-11ea-8ce3-001e677affd5:1-26" >
```

  - This will be seen while the resharding switches between the old and new
    shards.  Note the transition of the VGTID from a single shard to multiple
    shards.  These transactions have no row events, because they are
    transactions outside of the keyspace `customer` (operations on the 
    `_vt` sidecar database, to be precise).
  - Here is the setup of the sharded keyspace now:

```
$ vtctlclient -server localhost:15999 ListAllTablets | grep customer
zone1-0000000200 customer 0 master localhost:15200 localhost:17200 [] 2020-06-28T03:24:53Z
zone1-0000000201 customer 0 replica localhost:15201 localhost:17201 [] <null>
zone1-0000000202 customer 0 rdonly localhost:15202 localhost:17202 [] <null>
zone1-0000000300 customer -80 master localhost:15300 localhost:17300 [] 2020-06-28T03:31:12Z
zone1-0000000301 customer -80 replica localhost:15301 localhost:17301 [] <null>
zone1-0000000302 customer -80 rdonly localhost:15302 localhost:17302 [] <null>
zone1-0000000400 customer 80- master localhost:15400 localhost:17400 [] 2020-06-28T03:31:13Z
zone1-0000000401 customer 80- replica localhost:15401 localhost:17401 [] <null>
zone1-0000000402 customer 80- rdonly localhost:15402 localhost:17402 [] <null>
```

  - Note that you have the new shards (`-80` and `80-`), but the old
    shard (`0`) is still available.
  - Now, let's insert some more records to show that the stream still works
    after resharding:

```
$ mysql -A -u root -P 15306 -h 127.0.0.1 customer
.
.
mysql> insert into customer (customer_id, email) values (10, "1234");
Query OK, 1 row affected (0.01 sec)

mysql> insert into corder (order_id, customer_id, sku, price) values (25, 108, 123, "124445");
Query OK, 1 row affected (0.01 sec)
```

  - In the vStream client terminal, you will see something like:

```
Event log timestamp: 1593315270 --> 2020-06-28 03:34:30 +0000 UTC
+-------------------+--------+-------------+-------+
|       table       |   op   | customer_id | email |
+-------------------+--------+-------------+-------+
| customer.customer | INSERT |          10 |  1234 |
+-------------------+--------+-------------+-------+
VGTID after event:  shard_gtids:<keyspace:"customer" shard:"-80" gtid:"MySQL56/9e9047d4-b8ef-11ea-80a2-001e677affd5:1-29" > shard_gtids:<keyspace:"customer" shard:"80-" gtid:"MySQL56/b8b9ba74-b8ef-11ea-8ce3-001e677affd5:1-26" > 

Event log timestamp: 1593315275 --> 2020-06-28 03:34:35 +0000 UTC
+-----------------+--------+----------+-------------+-----+--------+
|      table      |   op   | order_id | customer_id | sku | price  |
+-----------------+--------+----------+-------------+-----+--------+
| customer.corder | INSERT |       25 |         108 | 123 | 124445 |
+-----------------+--------+----------+-------------+-----+--------+
VGTID after event:  shard_gtids:<keyspace:"customer" shard:"-80" gtid:"MySQL56/9e9047d4-b8ef-11ea-80a2-001e677affd5:1-29" > shard_gtids:<keyspace:"customer" shard:"80-" gtid:"MySQL56/b8b9ba74-b8ef-11ea-8ce3-001e677affd5:1-27" > 
```

  - Now, stop the vStream test client (CTRL-C)
  - Start it again with new arguments, referring to the original shard (`0`)
    and the very first GTID we saw during the initial output:

```
$ go run vstream_test_client.go -vtgate=localhost:15991 -keyspace=customer -tablet_type=replica -pos='[ {"shard":"0", "gtid":"MySQL56/d714417e-b8ee-11ea-ad61-001e677affd5:1-36"} ]'
```

  - You should see *all* the same events after the first `customer.corder`
    insert above being replayed (again).
  - Note that if you had run `306_down_shard_0.sh` before this, you would
    *not* have been able to resume from this original position, since the
    source shard for the pre-resharding events would have been unavailable.
  - You can also stop the vStream client, and start it from the resharding
    point, this will be your new minimum resynchronization point after you
    remove the `0` shard. In this case, you will only see the events after
    that point:

```
$ go run vstream_test_client.go -vtgate=localhost:15991 -keyspace=customer -tablet_type=replica -pos='[ {"shard":"-80", "gtid":"MySQL56/9e9047d4-b8ef-11ea-80a2-001e677affd5:1-28"}, {"shard":"80-", "gtid":"MySQL56/b8b9ba74-b8ef-11ea-8ce3-001e677affd5:1-25"} ]'
vtgate connecting to: localhost:15991
vgtid: shard_gtids:<keyspace:"customer" shard:"-80" gtid:"MySQL56/9e9047d4-b8ef-11ea-80a2-001e677affd5:1-28" > shard_gtids:<keyspace:"customer" shard:"80-" gtid:"MySQL56/b8b9ba74-b8ef-11ea-8ce3-001e677affd5:1-25" >
tablet_type: REPLICA


Event log timestamp: 1593315092 --> 2020-06-28 03:31:32 +0000 UTC
VGTID after event:  shard_gtids:<keyspace:"customer" shard:"-80" gtid:"MySQL56/9e9047d4-b8ef-11ea-80a2-001e677affd5:1-28" > shard_gtids:<keyspace:"customer" shard:"80-" gtid:"MySQL56/b8b9ba74-b8ef-11ea-8ce3-001e677affd5:1-26" > 

Event log timestamp: 1593315275 --> 2020-06-28 03:34:35 +0000 UTC
+-----------------+--------+----------+-------------+-----+--------+
|      table      |   op   | order_id | customer_id | sku | price  |
+-----------------+--------+----------+-------------+-----+--------+
| customer.corder | INSERT |       25 |         108 | 123 | 124445 |
+-----------------+--------+----------+-------------+-----+--------+
VGTID after event:  shard_gtids:<keyspace:"customer" shard:"-80" gtid:"MySQL56/9e9047d4-b8ef-11ea-80a2-001e677affd5:1-28" > shard_gtids:<keyspace:"customer" shard:"80-" gtid:"MySQL56/b8b9ba74-b8ef-11ea-8ce3-001e677affd5:1-27" > 

Event log timestamp: 1593315270 --> 2020-06-28 03:34:30 +0000 UTC
+-------------------+--------+-------------+-------+
|       table       |   op   | customer_id | email |
+-------------------+--------+-------------+-------+
| customer.customer | INSERT |          10 |  1234 |
+-------------------+--------+-------------+-------+
VGTID after event:  shard_gtids:<keyspace:"customer" shard:"-80" gtid:"MySQL56/9e9047d4-b8ef-11ea-80a2-001e677affd5:1-29" > shard_gtids:<keyspace:"customer" shard:"80-" gtid:"MySQL56/b8b9ba74-b8ef-11ea-8ce3-001e677affd5:1-27" > 
```

  - You can verify this by stopping the vStream client, and running
    from the Vitess `examples/local` directory:

```
$ ./306_down_shard_0.sh
```

  - And now running the vStream client from the same position as
    above, verifying that all the events post-resharding are
    still available.
  - As mentioned previously, trying now to run the vStream client
    from the original (shard `0` based) position will fail, because
    shard `0` is no longer available, i.e.:

```
$ go run vstream_test_client.go -vtgate=localhost:15991 -keyspace=customer -tablet_type=replica -pos='[ {"shard":"0", "gtid":"MySQL56/d714417e-b8ee-11ea-ad61-001e677affd5:1-36"} ]'
vtgate connecting to: localhost:15991
vgtid: shard_gtids:<keyspace:"customer" shard:"0" gtid:"MySQL56/d714417e-b8ee-11ea-ad61-001e677affd5:1-36" > 
tablet_type: REPLICA

remote error: Code: UNAVAILABLE
rpc error: code = Unavailable desc = target: customer.0.replica: no valid tablet
 at shard_gtids:<keyspace:"customer" shard:"0" gtid:"MySQL56/d714417e-b8ee-11ea-ad61-001e677affd5:1-36" > , retrying in 1s
.
.
etc.
```
