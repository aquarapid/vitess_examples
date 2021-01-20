# VTTablet connection pools and sizing:

VTTablet uses a variety of connection pools to connect to MySQLd.
Most of these can be controlled by vttablet options.  Note that almost 
all of these pools are **not** fixed size connection pools, and will grow
on demand to the maximum configured sizes;  and then potentially slowly
shrink back down again.

One thing to note is that each of these pools do not use unique MySQL
usernames, so it can be hard from a MySQL processlist to distinguish
between different pool connections.  Consult the `_active` pool metrics
(e.g. ``vttablet_dba_conn_pool_active`) as the authoritative resource on
how many MySQL protocol connections are in use for each pool.

## Pools:

* **transaction_pool** and **found_rows_pool**:
  * Max size (for each) controlled by:  `-queryserver-config-transaction-cap` (default 20)
  * metric:  `vttablet_transaction_pool_capacity`
  * metric:  `vttablet_found_rows_pool_capacity`
  * Used by transaction engine to manage transactions that require
  a dedicated connection.  The main pool for this use the **transaction_pool**.
  The **found_rows_pool** is dedicated for connections where the client is 
  using the `CLIENT_FOUND_ROWS` option (i.e. the affected rows variable returned
  by the MySQL protocol becomes the number of rows matched by the `WHERE`
  clause instead)
    

* **conn_pool**:
  * Max size controlled controlled by:  `-queryserver-config-pool-size` (default 16)
  * metric:  `vttablet_conn_pool_capacity`
  * Used as the vttablet query engine "normal" (non-streaming) connections pool.

* **stream_conn_pool**:
  * Max size controlled by:  `-queryserver-config-stream-pool-size`     (default 200)
  * metric:  `vttablet_stream_conn_pool_capacity`
  * Used as vttablet query engine streaming connections pool. All streaming
  queries that are not transactional should use this pool.

* **dba_conn_pool**:
  * Max size controlled by:  `-dba_pool_size`                           (default 20)
  * metric:  `vttablet_dba_conn_pool_capacity`
  * Used by vttablet `ExecuteFetchAsDba` RPC.  Also used implicitly for
  various internal Vitess maintenance tasks (e.g. schema reloads, etc.)

* **app_conn_pool**:
  * Max size controlled by:  `-app_pool_size`                           (default 40)
  * metric:  `vttablet_app_conn_pool_capacity`
  * Used by vttablet `ExecuteFetchAsApp` RPC.

* **tx_read_pool**:
  * Hardcoded                                                           (default 3)
  * metric:  `vttablet_tx_read_pool_capacity`
  * Used in the (non-default) TWOPC `transaction_mode` for metadata state
  management.  This pool will always be empty unless TWOPC is used.

* Pools associated with online DDL:
  * **online_ddl_executor_pool**:
    * Hardcoded                                                         (default 3)
    * Used in Online DDL to during the actual process of running gh-ost or pt-osc.
  * **table_gc_pool**:
    * Hardcoded                                                         (default 2)
    * Used in Online DDL to purge/evac/drop origin tables after Online
    DDL operations from them have been completed.
  * **throttler_pool**:
    * Hardcoded                                                         (default 2)
    * Used in Online DDL to measure/track the master -> replica lag, and
    adjust the DDL copy speed accordingly.

