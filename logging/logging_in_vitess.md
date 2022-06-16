# Logging in Vitess

Vitess components, like most server components, can produce two main types of
logs:
  * "Server" logs, detailing:
    * Information about or changes in the server state
    * Warnings
    * Errors
  * "Query" logs, a representation of the requests sent to the component by
  clients.

Some of the Vitess components that do not serve user queries (e.g. `vtctld`)
only generate "server" logs, and do not record "query" logs.


## Server logs

TODO

## Query logs

As mentioned before, the main components that support query logging are
`vtgate` and `vttablet`.  Query logs can be obtained in 3 ways from these
components:
  * Enabling query logging to a file using the appropriate commandline
  parameters for the component. This requires a restart of the component to
  enable/disable.
  * Using the component's web UI to display the logs in nicely formatted HTML
  output.
  * Accessing the low-level querylog endpoint via the component's web port
  and redirecting the output to a file.

The 2nd and 3rd options above are useful for debugging, i.e. where you do not
want permanent logging to a file (which carries a significant performance
penalty on a busy `vtgate` or `vttablet`), but still want to see the actual
client queries being made to a component when necessary.

There are also additional options to control the extent of the logging, e.g.:
  * the redaction of potentially sensitive queries; to conform to compliance
  requirements
  * only log queries that have specific tags in their content
  * other options to limit the size of queries logged, size of errors logged, etc.

### Query logging to a file

For both `vtgate` and `vttablet` the commandline option to enable query logging
to a file is the same:

```sh
-log_queries_to_file /path/to/filename.log
```

Notes:
  * The file will grow without limit.  You are reponsibile for rotating this
  log as necessary. Vitess does not include any mechanism to rotate
  query logs.
  * To help with external log rotation, `vtgate` and `vttablet` will reopen
  their query logfiles when they receive a SIGUSR2 signal.
  * As mentioned previously, logging to a file has a performance overhead,
  that can be significant at high QPS levels.  Both CPU usage and p99 latency
  will be impacted, often in the range of 10% at very high QPS.
  * The logfile format is essentially tab-delimited CSV (also known as TSV).
  It is conveniently `awk`-able and `grep`-able.

### Streaming query logs from the component

The same query log content as via `-log_queries_to_file` can be
streamed from the `vtgate` or `vtgate` web port, either locally or remotely,
by using a normal HTTP client (e.g. `curl`). This can be a convenient way to
get querylogs for debugging without having to enable query logging to a file,
which would need a component restart, because of the commandline parameter
change required.

The format if the log output is identical to that to the query logging to a
file (i.e. tab-delimited CSV).  Because it is streamed via the HTTP socket,
it is relatively easy to create a pipeline that takes this stream as input
and then streams the logs into a different remote logging system (e.g.
syslog, filebeats, etc.).  You can also use curl locally to redirect the
stream to a file for as long as you need to capture it, potentially redirecting
via `gzip` or `zstd` if disk consumption is a concern.

Example (for vtgate or vttablet):
 * Capture query logs for 60 seconds to the file `/var/tmp/temp_queries.tsv`:
 ```sh
$ timeout 60 curl http://127.0.0.1:xxxx/debug/querylog > /var/tmp/temp_queries.tsv
 ```
 * `xxxx` is the web UI port of the `vtgate` or `vttablet` component.
  In the "local" examples in the Vitess distribution/source, the `vtgate` port
  would be `15001` and the vttablet port would be `15100`, `15101`, etc.
  depending on which `vttablet` instance you want to retrieve logs from.
  The web UI port is specified to the `vtgate` and `vttablet` components
  using the `-port` parameter.


### Log format layouts

TODO:  text (and json?)

The file/streaming logging output supports two formats, specified via the
`-querylog-format` option:
 * `text` - the default
 * `json`

The `text` format is structured as follows for **vtgate** query logs,
with the previously mentioned tab delimiters:
 * Method -
 * Client address and port
 * VTGate username
 * Immediate Caller - TODO
 * Effective Caller - TODO
 * Query start time (in `vtgate`)  - with microsecond precision
 * Query end time (in `vtgate`) - with microsecond precision
 * Total query time (since entering `vtgate` to completion) - in fractional seconds, with microsecond precision
 * Query planning time - in fractional seconds, with microsecond precision
 * Query execution time - time taken to execute query, including downstream time to vttablet, etc.  in fractional seconds, with microsecond precision
 * Query commit time - time the commit phase (or phases if multi-shard) of the query took. in fractional seconds, with microsecond precision

For **vttablet** query logs, the `text` format is structured as follows,
again with tab delimiters:
 * Method - gRPC method that resulted in this log line, e.g.:
   * `Execute`
   * `StreamExecute` - if you are using `set workload=olap`
   * `Begin`
   * `Commit`
   * etc.
 * CallInfo - gRPC call info; typically the gRPC call source IP/port (vtgate
 source IP/port for the gRPC client), plus the `vttablet` gRPC method URI,
 e.g.:
   * `/queryservice.Query/Execute`
   * `/queryservice.Query/StreamExecute`
   * `/queryservice.Query/Begin`
   * `/queryservice.Query/Commit`
   * `/queryservice.Query/BeginStreamExecute`
   * `/queryservice.Query/Commit`
   * `/queryservice.Query/Rollback`
   * etc.
 * Username - TODO (always `gRPC`?)
 * Immediate Caller - `vtgate` caller context sent from the gRPC client
 (`vtgate`).  This will typically be the `vtgate` MySQL user used by the
 application if using the MySQL protocol.
 * Effective Caller - TODO
 * Query start time (in `vttablet`) - with microsecond precision
 * Query end time - (in `vttablet`) - kwith microsecond precision
 * Total query time (since entering `vttablet`, until returning last byte to `vtgate`) - in fractional seconds, with microsecond precision
 * Plan Type - The type of query this was classified as by the `vttablet` planbuilder.
 This is a large set, but the common values are strings like:
   * `Select`
   * `Insert`
   * `Update`
   * `Delete`
   * etc.
 * Original SQL - Original SQL as received by `vttablet`; note that this is
 **not** the same as the original SQL sent by the application to `vtgate`;
 since the SQL has already potentially been rewritten and/or expanded by
 `vtgate`.
 * BindVars - Map of bind variables and values
 * Queries - TODO
 * Rewritten SQL - SQL after being rewritten by `vttablet`, i.e. the SQL
 that `vttablet` attempted to execute.
 * Query Sources - indicator of the source of the `vttablet` query. Typical
 values: `mysql` (meaning client via `vtgate`) and `consolidator`
 * MySQL time - Time spent in MySQL, i.e. how long it took for the query
 to return after it was sent to MySQL.
 * Conn Wait Time - How long (in seconds with microsecond precision) the
 `vttablet` <-> MySQL query had to wait for a pool connection. This is
 always non-zero, but unless the connection pool has no free connections,
 should be relatively low.
 * Rows Affected
 * Transaction ID - if applicable
 * Response Size - in bytes
 * Error - if any

  

### Query logging in the web UI

TODO

### Additional logging control options

TODO:
 * `-redact-debug-ui-queries`
 * `-querylog-filter-tag`
 * `-log_queries`: syslog - does this still work?
 * `-sql-max-length-ui`
 * `-sql-max-length-errors`
 * `-stderrthreshold`
 * `-v`
 * `-keep_logs`
   * Only works for "server" logs, not "query" logs
 * `-keep_logs_by_mtime`
   * Only works for "server" logs, not "query" logs
 * `-log_rotate_max_size`
   * Only works for "server" logs, not "query" logs, which will grow without
     bound.
 * `-purge_logs_interval`
   * Only works for "server" logs, not "query" logs

  * other options to limit the size of queries logged, size of errors logged, etc.
