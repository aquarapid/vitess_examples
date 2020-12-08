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

```
-log_queries_to_file /path/to/filename.log
```

Note:
  * The file will grow without limit.  You are reponsibile for rotating this
  log as necessary. Vitess does not include any mechanism to rotate
  query logs.
  * To help with external log rotation, `vtgate` and `vttablet` will reopen
  their query logfiles when they receive a SIGUSR2 signal.
  * As mentioned previously, logging to a file has a performance overhead,
  that can be significant at high QPS levels.  Both CPU usage and p99 latency
  will be impacted, often in the range of 10% at very high QPS.

### Query logging in the web UI

TODO

### Streaming query logs from the component

TODO

