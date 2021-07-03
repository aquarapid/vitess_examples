By default, when launching Vitess in a Docker container the default memory
consumption might be rather high.

This can be seen from the following example:

```
$ docker run -p 15000:15000 -p 15001:15001 -p 15991:15991 -p 15999:15999 -it vitess/local
add /vitess/global
add /vitess/zone1
add zone1 CellInfo
ERROR: logging before flag.Parse: E0703 01:02:19.395150      38 syslogger.go:149] can't connect to syslog
W0703 01:02:19.552014      38 vtctl.go:90] cannot connect to syslog: Unix syslog delivery error
etcd start done...
Starting vtctld...
Starting MySQL for tablet zone1-0000000100...
ERROR: logging before flag.Parse: E0703 01:02:19.808492      74 syslogger.go:149] can't connect to syslog
Starting vttablet for zone1-0000000100...
HTTP/1.1 200 OK
Date: Sat, 03 Jul 2021 01:02:28 GMT
Content-Type: text/html; charset=utf-8

Starting MySQL for tablet zone1-0000000101...
ERROR: logging before flag.Parse: E0703 01:02:28.926715     744 syslogger.go:149] can't connect to syslog
Starting vttablet for zone1-0000000101...
HTTP/1.1 200 OK
Date: Sat, 03 Jul 2021 01:02:36 GMT
Content-Type: text/html; charset=utf-8

Starting MySQL for tablet zone1-0000000102...
ERROR: logging before flag.Parse: E0703 01:02:36.768541    1373 syslogger.go:149] can't connect to syslog
Starting vttablet for zone1-0000000102...
HTTP/1.1 200 OK
Date: Sat, 03 Jul 2021 01:02:43 GMT
Content-Type: text/html; charset=utf-8

ERROR: logging before flag.Parse: E0703 01:02:43.785408    2002 syslogger.go:149] can't connect to syslog
I0703 01:02:44.967570    2043 main.go:67] I0703 01:02:44.966212 tablet_executor.go:277] Received DDL request. strategy=direct
I0703 01:02:45.066443    2043 main.go:67] I0703 01:02:45.066144 tablet_executor.go:277] Received DDL request. strategy=direct
I0703 01:02:45.185862    2043 main.go:67] I0703 01:02:45.185621 tablet_executor.go:277] Received DDL request. strategy=direct
New VSchema object:
{
  "tables": {
    "corder": {

    },
    "customer": {

    },
    "product": {

    }
  }
}
If this is not what you expected, check the input data (as JSON parsing will skip unexpected fields).
Waiting for vtgate to be up...
vtgate is up!
Access vtgate at http://0f2514ecc017:15001/debug/status
vitess@0f2514ecc017:/vt/local$ cat /sys/fs/cgroup/memory/memory.usage_in_bytes
1439567872
```

Note from the cgroup output that this default `docker/local` example setup,
which starts a single shard with 3 tablets (i.e. 3 MySQLd instances), consumes
about 1.4 GB of memory.  Depending on your needs (e.g. running testing with
the `docker/local` container), this might be excessive, especially if you
want to launch a multi-shard setup with many more tablets.

The bulk of the memory usage is from the actual MySQLd instance, and can be
reduced by adjusting the MySQL `my.cnf` parameters. To achieve
this, we are going to create a "supplemental" `my.cnf` configuration file
for the MySQLd instances, and pass it into the Docker container for Vitess\`
mysqlctl to apply when initializing the MySQLd instances.  These
parameters are not meant for any real-world usage, since they switch
of MySQL performance schema and reduce the size of the default buffers.
Note the `-v` and `-e` parameters passed to the `docker run` command:

```
$ cat /tmp/extra.cnf
innodb_buffer_pool_chunk_size=33554432
innodb_buffer_pool_size=33554432
key_buffer_size=1048576
performance_schema=OFF

$ docker run -p 15000:15000 -p 15001:15001 -p 15991:15991 -p 15999:15999 -v /tmp/extra.cnf:/tmp/extra.cnf -e EXTRA_MY_CNF=/tmp/extra.cnf -it vitess/local
add /vitess/global
add /vitess/zone1
add zone1 CellInfo
ERROR: logging before flag.Parse: E0703 01:05:27.910425      37 syslogger.go:149] can't connect to syslog
W0703 01:05:27.926419      37 vtctl.go:90] cannot connect to syslog: Unix syslog delivery error
etcd start done...
Starting vtctld...
Starting MySQL for tablet zone1-0000000100...
ERROR: logging before flag.Parse: E0703 01:05:27.967270      84 syslogger.go:149] can't connect to syslog
Starting vttablet for zone1-0000000100...
HTTP/1.1 200 OK
Date: Sat, 03 Jul 2021 01:05:35 GMT
Content-Type: text/html; charset=utf-8

Starting MySQL for tablet zone1-0000000101...
ERROR: logging before flag.Parse: E0703 01:05:35.440263     785 syslogger.go:149] can't connect to syslog
Starting vttablet for zone1-0000000101...
HTTP/1.1 200 OK
Date: Sat, 03 Jul 2021 01:05:43 GMT
Content-Type: text/html; charset=utf-8

Starting MySQL for tablet zone1-0000000102...
ERROR: logging before flag.Parse: E0703 01:05:43.252437    1463 syslogger.go:149] can't connect to syslog
Starting vttablet for zone1-0000000102...
HTTP/1.1 200 OK
Date: Sat, 03 Jul 2021 01:05:51 GMT
Content-Type: text/html; charset=utf-8

ERROR: logging before flag.Parse: E0703 01:05:51.444982    2145 syslogger.go:149] can't connect to syslog
I0703 01:05:52.433055    2187 main.go:67] I0703 01:05:52.431999 tablet_executor.go:277] Received DDL request. strategy=direct
I0703 01:05:52.545644    2187 main.go:67] I0703 01:05:52.545310 tablet_executor.go:277] Received DDL request. strategy=direct
I0703 01:05:52.655607    2187 main.go:67] I0703 01:05:52.655381 tablet_executor.go:277] Received DDL request. strategy=direct
New VSchema object:
{
  "tables": {
    "corder": {

    },
    "customer": {

    },
    "product": {

    }
  }
}
If this is not what you expected, check the input data (as JSON parsing will skip unexpected fields).
Waiting for vtgate to be up...
vtgate is up!
Access vtgate at http://c32c639c8cc2:15001/debug/status
vitess@c32c639c8cc2:/vt/local$ cat /sys/fs/cgroup/memory/memory.usage_in_bytes
794324992
```

As can be seen, we reduced the baseline memory usage for the 3 tablet example
container from 1.4 GB to around 800 MB.  This is with the default MySQL 5.7
version used for the `docker/local` image (at the time of writing, MySQL
`5.7.31`).  If the `docker/local` image is built with the MySQL 8.0 install
(currently `8.0.23`), the difference is a bit less dramatic (about
1.05 GB vs 1.6 GB), since MySQL 8.0 just has a larger base memory footprint.

