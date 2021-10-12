# Vitess comment query timeouts

In Vitess, individual queries are subject to query timeouts, typically set
by the `vttablet` option `-queryserver-config-query-timeout`.  Whole 
transactions are also subject to the `vttablet` timeout setting of
`-queryserver-config-transaction-timeout`.

However, for read (`SELECT`) queries, it is also possible to use a special
Vitess query comment format to set a lower timeout for certain queries, e.g.:

```
mysql> select /*vt+ QUERY_TIMEOUT_MS=1 */ sleep(1);
ERROR 1317 (70100): target: keyspace1.0.primary: vttablet: rpc error: code = DeadlineExceeded desc = context deadline exceeded
```

As indicated by the comment name (`QUERY_TIMEOUT_MS`), this timeout is in
milliseconds.

## Limitation/caveats:
  * Only works for `SELECT` queries.
  * Does not work when doing manual shard-targeting (see https://github.com/vitessio/vitess/issues/7031)
  * Cannot set a higher limit to evade the settings for `-queryserver-config-query-timeout`
    and/or `-queryserver-config-transaction-timeout`.
  * If your client strips comments before sending your SQL to the server, this
    option will obviously not work. For example, the MySQL client does this by
    default, unless you use the `-c` option.


