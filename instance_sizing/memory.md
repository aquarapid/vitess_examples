# Sizing memory for MySQL instances for use with Vitess

Unfortunately, to start with, the general guideline is "it depends".  However,
we want to be somewhat more helpful than that, so here are some details on
how you might think about sizing MySQL instance memory in Vitess relative
to data size and workload.

## Do you really care?

Obviously, if your database does not have high performance requirements,
it might not be necessary for any significant portion of your data to
be cached by MySQL to get acceptable performance.  On the other hand,
your buffer pool memory might be signicantly larger than the size of
your database, and so things are probably as fast as it is going to
get.  However, you probably would not be using Vitess if either of
these were the case, so read on.

## Why do we care?

If the MySQL buffer pool (the portion of memory set aside by a MySQL server
for caching and opeartions) does not contain the pages that we need to
satisfy the most frequent queries, every uncached read operation might need
to go to disk (probably several times for larger tables) to satisfy a
request.  Even in a world where you are running your servers on-prem, and
using fast NVMe storage, this could be expensive.  In the cloud world, where
disk access is slower, and IOPS may be limited, this is even more
important.

## What factors matter?

  * How much memory do you have available?
    * This might seem obvious, but not all the memory on a MySQL server is
    automatically available for use by the buffer pool.  It typically needs
    to be sized manually (although newer MySQL versions does have some
    auto-sizing features).  This size might be as little as 75% of the total
    memory on the server, and using more might result in swapping (bad) or
    OOMs (worse).  If your MySQL server is co-located with other components
    (e.g. `vttablet` in the case of Vitess), you will want to factor in the
    estimated memory usage for that as well.
  * Overall data size (how large are the parts of the data stored for every
  row in every table)
  * Overall **active** data size:
    * Luckily for many applications, not all data in every table is accessed
    very often.  And if older data is accessed, the expectations of the
    access time for that data might be relaxed.  This might allow you to
    (say) only have 10% of a table cached in memory, and still get very
    acceptable performance for that table.  You should estimate this for
    the major (i.e. large) entities (and associated tables) in your database.
    It is often easier to think about the amount of data that you might
    need well-cached by thinking of time instead of an immediate percentage.
    You can then convert something like "the last 3 days of data for table
    payments needs to be well cached" into something concrete, e.g.: "3 days
    of payments and associated data will be 20 GB per shard"
    * For some applications, the actual "wide" columns for some tables are
    extracted rather infrequently (but watch out for ORMs that just load up
    whole rows every time!), so you might be able to exclude these columns
    from your calculation, **but only if they are large enough and/or of
    a data type that is stored off-page in the table BTree**.
    * When estimating these numbers for active data, remember that the
    cachable unit in MySQL/InnoDB is a 16 kB page.  You could have a single
    row occupying 200 bytes on a page, and you would still need to cache
    the whole page in memory to access it without disk I/O.  This is why
    it is critically important that the primary key for large tables are
    clustered together in the BTree with regards to time, so that if you
    have one row on a page that you might want to cache, it is likely
    that the other rows are also of interest (to cache).  This is not
    always possible, however, and this is important to think about for tables
    like Vitess lookup vindex backing tables.
  * Do not forget indexes when estimating active data size:
    When displaying sizes for a table in MySQL via `show table status`,
    the primary key index is included in the data size, since MySQL uses a
    PK-clustered storage strategy.  It goes without saying that keeping
    this in memory is even more critical than keeping the actual row data
    in memory.  However, depending on your use-case, keeping at least a
    subset of your most important secondary indexes in memory may also be
    critical.  You can obtain per-secondary-index sizing statistics from
    the MySQL `information_schema` to make estimates here.
  * Beware full table scans:
    * While MySQL does have features to protect the contents of the buffer
    pool against queries that perform full table scans, this is not perfect.
    It is therefore assumed that you have already taken measures to eliminate
    or limit (maybe to a replica) queries that do large full table scans.
  * Consider the write rate and dirty page flushing for your use-case:
    * If you write large rows (or large numbers of smaller rows), a significant
    portion of your buffer pool may be used to buffer the dirty pages, hoping
    to accumulate additional changes to these pages before they are written
    to disk.  If this includes your use-case, you may want to ensure that
    you have already:
      * Sized the InnoDB log files appropriately for the write load
      * Adjusted the InnoDB capacity settings appropriately for your
      storage, to allow the flushing to proceed fast enough as to not
      become a bottleneck.
      * Depending on your memory requirements for read caching, you may
      wnat to limit the portion of the buffer pool that can be dirtied
      by writes by limiting `innodb_max_dirty_pages_pct`
  * Don't forget the other users of your buffer pool memory:
    * Even though it is called the InnoDB buffer pool, there are other
    features of InnoDB that use (potentially significant) parts
    of this memory, and you should keep in mind:
      * Adaptive Hash Index (AHI):
        * We will not go into a full breakdown of this, other than to say
    that the amount of memory used by the AHI is unpredictable, and
    **cannot be controlled**, other than by disabling it entirely.
    You may want to experiment with examining the metrics associated
    with it closely, and then with disabling it.  For larger MySQL
    instances with large tables that significantly exceed the
    size of available buffer pool memory, this can often be a
    performance win.
      * Change buffer:
        * MySQL will try and defer updates to secondary keys using the
    change buffer, again to reduce the I/O associated with these
    updates.  Unless you have very extensive secondary indexes and
    a high write rate to those tables, the amount of memory used
    by the change buffer is probably not significant enough to
    worry about.

## Guidelines

Now that we have covered the caveats, here are some more concrete guidelines:

 * At a minimum, have enough buffer pool memory to cache the "hot" parts
 of your important table (PK + row data).  Add a margin of 25% to this number.
 * Add additional memory to your estimate from the secondary index sizes,
 again preferably the hot parts plus 25%.
 * If you have large columns in some important tables that are not accessed
 on every query, you may want to consider if they should not be stored off-page,
 either explicitly (separate table), or implicitly (by selecting the correct data
 type).
   * A similar strategy might be considered for whole sets of "cold" columns
   in a hot table.
 * Try to estimate what proportion of your buffer pool is dirty at any given
 time.  If these dirty pages are also the "hot" parts of your tables, that
 is great.  If not, you have to include this in your estimates of memory.
 * For sharded Vitess, external secondary indexes ("lookup vindexes") are
 common.  Typically these take the form of a two (although more are possible if
 the vindex is multi-column) column table, mapping a key to a `keyspace_id`.
 Do not neglect to factor these lookup tables, and the impact of its access
 pattern into your estimates.
 * For sharded Vitess, you will need to make your estimates on a per-shard
 basis, since every shard is hosted by its own MySQL instance(s).
 * You can use the content of `information_schema.innodb_cached_indexes` to
 estimate how many pages are actually being used per index.  Keep in mind
 that your initial setup might be memory-rich, and thus the number of cached
 index pages will tend to grow for a long time after a MySQL server restart.
 It is therefore appropriate to sample this table after a certain amount of
 time has elapsed since a server restart, and the buffer is "warm enough"
 and performance acceptable.
 * The raw values RE indexes sizes can be obtained from the table
 `mysql.innodb_index_stats`;  this includes the primary and secondary keys.
 Remember that the "primary key" size includes the data rows.
 * Note that index statistics in MySQL/InnoDB is approximate (via sampling),
 and therefore most of the row estimates in the various table and index-related
 metadata that MySQL reports is at best "in the ballpark", and can be
 very significantly off for certain cases (very large tables;  tables with
 heavily skewed data).  To combat this, ensure that you:
   * Have the InnoDB page sampling settings adjusted to sample more pages
   than the defaults
   * The table statistics has been updated recently (usually via
   `analyze table`).  You can see when this last happened (either explicitly
   or implicitly) by inspecting the `last_update` timestamp in the
   `mysql.innodb_index_stats` table.  Beware that `analyze table` can be
   disruptive on very busy MySQL instanes, so run during quiet periods.
 * Not useful for sizing, but in general Vitess MySQL instance operations
 that are related to memory / buffer pool:
   * Configure the `innodb_buffer_pool_dump_at_shutdown` feature; and bump the
   `innodb_buffer_pool_dump_pct` to 100.  This will dump the IDs all the
   InnoDB pages in the buffer pool to disk on shutdown, allowing them to
   be loaded in the background after restart.  This will help when rolling
   instances for upgrades or other maintenance.

## Examples:

 * Obtaining the table (i.e. clustered PRIMARY index) and secondary index
 on-disk sizes for a table:
```
mysql> select table_name, index_name, round(stat_value*16384/1024/1024) as size_MiB 
from mysql.innodb_index_stats where table_name = "sbtest1" and database_name = "test1" 
and stat_name = "size";
+------------+------------+----------+
| table_name | index_name | size_MiB |
+------------+------------+----------+
| sbtest1    | PRIMARY    |     6428 |
| sbtest1    | k_1        |      446 |
+------------+------------+----------+
```
 * Combining the above with the statistics about how much of the indexes
 are currently cached, keeping in mind that some indexes might not show
 up in this query if none of their pages are in cache.  This might occur
 when either the index has never been used, or the index page was in cache,
 but was evicted because of memory pressure elsewhere:
```
mysql> select innodb_tables.name AS table_name, innodb_indexes.name AS index_name, 
round(stat_value*16384/1024/1024) as size_MiB, 
round(innodb_cached_indexes.n_cached_pages*16384/1024/1024) as cached_MiB 
FROM information_schema.innodb_cached_indexes, information_schema.innodb_indexes, 
information_schema.innodb_tables, mysql.innodb_index_stats 
WHERE database_name = "test1" and innodb_index_stats.table_name = "sbtest1" 
and innodb_cached_indexes.index_id = innodb_indexes.index_id and 
innodb_indexes.table_id = innodb_tables.table_id and 
innodb_tables.name = concat(innodb_index_stats.database_name, "/", innodb_index_stats.table_name) 
and stat_name = "size" and innodb_index_stats.index_name = innodb_indexes.name 
order by table_name, size_MiB desc;
+---------------+------------+----------+------------+
| table_name    | index_name | size_MiB | cached_MiB |
+---------------+------------+----------+------------+
| test1/sbtest1 | PRIMARY    |     6428 |         25 |
| test1/sbtest1 | k_1        |      446 |         17 |
+---------------+------------+----------+------------+
```
 * For reference, the table used above is a standard sysbench test table,
 with definition:
```
CREATE TABLE `sbtest1` (
  `id` int NOT NULL AUTO_INCREMENT,
  `k` int NOT NULL DEFAULT '0',
  `c` char(120) NOT NULL DEFAULT '',
  `pad` char(60) NOT NULL DEFAULT '',
  PRIMARY KEY (`id`),
  KEY `k_1` (`k`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
```
