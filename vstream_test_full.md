We are going to demonstrate streaming the contents as well as changes
from a Vitess database using the vStream test client 
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
$ ./101_initial_cluster.sh
```

  - Now you have an unsharded `commerce` keyspace setup.
  - In the previous example, we sharded an unsharded keyspace, but this
time we are just going to insert some data in the unsharded keyspace,
and show how you can use vStream to get all the rows from the tables 
in a database and then switch transparently to changes to the database.
  - First, let's insert some rows into the database:

```
$ mysql -A -u root -P 15306 -h 127.0.0.1 commerce
.
.
mysql> insert into product (sku,description, price) values ("12345", "1234", 100);
Query OK, 1 row affected (0.02 sec)

mysql> insert into product (sku,description, price) values ("123456", "1234", 100);
Query OK, 1 row affected (0.01 sec)
```

  - Now, run (in a separate terminal), from this repo:

```
$ go run vstream_test_client.go -vtgate=localhost:15991 -keyspace=commerce -tablet_type=replica -pos='[ {"shard":"0", "gtid":""} ]'
vtgate connecting to: localhost:15991
vgtid: shard_gtids:<keyspace:"commerce" shard:"0" > 
tablet_type: REPLICA


Event log timestamp: 0 --> 1970-01-01 00:00:00 +0000 UTC
VGTID after event:  shard_gtids:<keyspace:"commerce" shard:"0" gtid:"MySQL56/6b668cf6-c64e-11ea-9245-001e677affd5:1-17" > 

Event log timestamp: 0 --> 1970-01-01 00:00:00 +0000 UTC
VGTID after event:  shard_gtids:<keyspace:"commerce" shard:"0" gtid:"MySQL56/6b668cf6-c64e-11ea-9245-001e677affd5:1-17" > 

Event log timestamp: 0 --> 1970-01-01 00:00:00 +0000 UTC
+------------------+--------+-------+-------------+-------+
|      table       |   op   |  sku  | description | price |
+------------------+--------+-------+-------------+-------+
| commerce.product | INSERT | 12345 |        1234 |   100 |
+------------------+--------+-------+-------------+-------+
+------------------+--------+--------+-------------+-------+
|      table       |   op   |  sku   | description | price |
+------------------+--------+--------+-------------+-------+
| commerce.product | INSERT | 123456 |        1234 |   100 |
+------------------+--------+--------+-------------+-------+
VGTID after event:  shard_gtids:<keyspace:"commerce" shard:"0" gtid:"MySQL56/6b668cf6-c64e-11ea-9245-001e677affd5:1-17" 4:"\n\aproduct\x1a\r\"\v\n\x01\f\x12\x06123456" > 

Event log timestamp: 0 --> 1970-01-01 00:00:00 +0000 UTC
VGTID after event:  shard_gtids:<keyspace:"commerce" shard:"0" gtid:"MySQL56/6b668cf6-c64e-11ea-9245-001e677affd5:1-17" > 
```

  - First, note that we are specifying a gtid value of `""` (empty string).
This is shorthand for: "I want all rows and events from the database since
the beginning of time"
  - Second, note that we are specifying a specific shard (`"0"` in this case).
This is intentional, this "full data" mode only works when you stream by
specific shard.
  - Note the output. What we are seeing here is somewhat subtle:
    - We are not actually seeing a replay from the binary logs since 
"the beginning of time", we are seeing:
      - First a copy of all rows from all tables, emited as artificial row
events, followed by
      - a transition to normal binary row events, as they come in.
  - In a different terminal, connect to use `commerce` database and insert some more data:

```
$ mysql -A -u root -P 15306 -h 127.0.0.1 commerce
.
.
mysql> insert into product (sku,description, price) values ("1234567", "1234", 100);
Query OK, 1 row affected (0.01 sec)
```

  - You should now see the new row event in the vStream client terminal:

```
Event log timestamp: 1594785509 --> 2020-07-15 03:58:29 +0000 UTC
+------------------+--------+---------+-------------+-------+
|      table       |   op   |   sku   | description | price |
+------------------+--------+---------+-------------+-------+
| commerce.product | INSERT | 1234567 |        1234 |   100 |
+------------------+--------+---------+-------------+-------+
VGTID after event:  shard_gtids:<keyspace:"commerce" shard:"0" gtid:"MySQL56/6b668cf6-c64e-11ea-9245-001e677affd5:1-18" > 
```

  - Let's stop (CTRL-C) the vStream test client.
  - Now, let's delete some data to show that we are not just seeing
a replay of the log events:

```
$ mysql -A -u root -P 15306 -h 127.0.0.1 commerce
.
.
mysql> delete from product where sku = "12345";
Query OK, 1 row affected (0.02 sec)
```

  - Now, let's start the vStream client again, and stream from the beginning.
We will see this time that the insert and delete of the `sku = "12345"` is
completely missing, since we're just copying the rows from the database as
they appear at the time the vStream starts:

```
$ go run vstream_test_client.go -vtgate=localhost:15991 -keyspace=commerce -tablet_type=replica -pos='[ {"shard":"0", "gtid":""} ]'
vtgate connecting to: localhost:15991
vgtid: shard_gtids:<keyspace:"commerce" shard:"0" > 
tablet_type: REPLICA


Event log timestamp: 0 --> 1970-01-01 00:00:00 +0000 UTC
VGTID after event:  shard_gtids:<keyspace:"commerce" shard:"0" gtid:"MySQL56/6b668cf6-c64e-11ea-9245-001e677affd5:1-19" > 

Event log timestamp: 0 --> 1970-01-01 00:00:00 +0000 UTC
VGTID after event:  shard_gtids:<keyspace:"commerce" shard:"0" gtid:"MySQL56/6b668cf6-c64e-11ea-9245-001e677affd5:1-19" > 

Event log timestamp: 0 --> 1970-01-01 00:00:00 +0000 UTC
+------------------+--------+--------+-------------+-------+
|      table       |   op   |  sku   | description | price |
+------------------+--------+--------+-------------+-------+
| commerce.product | INSERT | 123456 |        1234 |   100 |
+------------------+--------+--------+-------------+-------+
+------------------+--------+---------+-------------+-------+
|      table       |   op   |   sku   | description | price |
+------------------+--------+---------+-------------+-------+
| commerce.product | INSERT | 1234567 |        1234 |   100 |
+------------------+--------+---------+-------------+-------+
VGTID after event:  shard_gtids:<keyspace:"commerce" shard:"0" gtid:"MySQL56/6b668cf6-c64e-11ea-9245-001e677affd5:1-19" 4:"\n\aproduct\x1a\x0e\"\f\n\x01\x0e\x12\a1234567" > 

Event log timestamp: 0 --> 1970-01-01 00:00:00 +0000 UTC
VGTID after event:  shard_gtids:<keyspace:"commerce" shard:"0" gtid:"MySQL56/6b668cf6-c64e-11ea-9245-001e677affd5:1-19" >
```

  - Lastly, note the timestamps of `0`; this is one way to distinguish
the artificial row events from the copying stage from "real" or "live"
row events. If you look back, you will see when we originally inserted
the `sku = 1234567` record, it had a "real" timestamp of:

```
Event log timestamp: 1594785509 --> 2020-07-15 03:58:29 +0000 UTC
```

  - However, upon the next run of the vStream client when it just copied
this row, the timestamp for the "same" row event was:

```
Event log timestamp: 0 --> 1970-01-01 00:00:00 +0000 UTC
```

  - So the original timestamps are not preserved for the copy phase.

