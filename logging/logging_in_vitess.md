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
  * Enabling query logging to a file using the appropriate commandline parameters
  for the component. This requires a restart of the component.
  * Using the component's web UI to display the logs in nicely formatted HTML output
  * Accessing the low-level querylog endpoint via the component's web port
  and redirecting the output to a file.

The 2nd and 3rd options above are useful for debugging, i.e. where you do not
want permanent logging to a file (which carries a performance penalty), but
still want to see the actual client queries being made to a component when
necessary.

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
  * The logfile format is essentially tab-delimited CSV (or TSV). It is
  coveniently awk-able and grep-able.

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
stream to a file for as long as you need to capture it.

Example (for vtgate or vttablet):
 * Capture query logs for 60 seconds to the file `/var/tmp/temp_queries.tsv`:
 ```sh
$ timeout 60 curl http://127.0.0.1:xxxx/debug/querylog > /var/tmp/temp_queries.tsv
 ```
 * `xxxx` is the web UI port of the `vtgate` or `vttablet` component.
  In the "local" examples in the Vitess distribution/source, the `vtgate` port
  would be `15001` and the vttablet port would be `15100`, `15001`, etc.
  depending on which `vttablet` instance you want to retrieve logs from.
  The web UI port is specified to the `vtgate` and `vttablet` components
  using the `-port` parameter.


### Log format layouts

TODO:  text (and json?)

The file/streaming logging output supports two formats:
 * `text` - the default
 * `json`

The `text` format is structured as follows for **vtgate** query logs,
with the previously mentioned tab delimiters:
 * Method -
 * Client address and port
 * VTGate username
 * Immediate Caller - TODO
 * Effective Caller - TODO
 * Query start time - with microsecond precision
 * Query end time - with microsecond precision
 * Total query time - in fractional seconds, with microsecond precision
 * Query planning time - in fractional seconds, with microsecond precision
 * Query execution time - time taken to execute query, including downstream time to vttablet, etc.  in fractional seconds, with microsecond precision
 * Query commit time - time the commit phase (or phases if multi-shard) of the query took. in fractional seconds, with microsecond precision

  

### Query logging in the web UI

TODO

### Additional logging control options

TODO:
 * `-querylog-format`
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
