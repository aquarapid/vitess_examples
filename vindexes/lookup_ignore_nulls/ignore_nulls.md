# The `ignore_nulls` option for lookup Vindex types

For `consistent_lookup` and `lookup` Vindex types (and their unique counterparts), you
cannot insert a `NULL` value for the lookup table column (also called the
"input"). Let's use `vtexplain` and this `vschema.json` VSchema file as
an example:

```
{
    "ks1": {
        "sharded": true,
        "vindexes": {
            "xxhash": {
                "type": "xxhash"
            },
            "lookup": {
                "type": "consistent_lookup_unique",
                "params": {
                    "table": "lookup1",
                    "from": "c2",
                    "to": "keyspace_id"
                },
                "owner": "t1"
            }
        },
        "tables": {
            "t1": {
                "column_vindexes": [
                    {
                        "column": "c1",
                        "name": "xxhash"
                    },
                    {
                        "column": "c2",
                        "name": "lookup"
                    }
                ]
            },
            "lookup1": {
                "column_vindexes": [
                    {
                        "column": "c2",
                        "name": "xxhash"
                    }
                ]
            }
        }
    }
}
```

The corresponding schema file (`schema.sql`):

```
CREATE TABLE t1 (
    c1 BIGINT NOT NULL,
    c2 BIGINT DEFAULT NULL,
    PRIMARY KEY (c1)
) ENGINE=Innodb;


CREATE TABLE lookup1 (
    c2 BIGINT NOT NULL,
    keyspace_id binary(8),
    UNIQUE KEY (c2)
) ENGINE=Innodb;
```

Now, if we use this combination and try to insert a row into table `t1` with a
`NULL` value for the `c2` column, we will get an error, e.g.:

```
$ vtexplain -schema-file schema.sql -vschema-file vschema.json -shards 8 -sql 'insert into t1 (c1, c2) values (1, NULL);'
ERROR: vtexplain execute error in 'insert into t1 (c1, c2) values (1, NULL)': execInsertSharded: getInsertShardedRoute: lookup.Create: input has null values: row: 0, col: 0
```

i.e. `vtexplain` is saying we are not allowed to use `NULL` input values for a
lookup Vindex (any of the various lookup Vindex types).  However, there are
cases in which you may want to insert a row with a `NULL` value for the lookup
table input column (`c2` in this case) into the owner table. In this case, we
do not want a matching row inserted into the lookup table, since it may break
the uniqueness constraint of the lookup table.  To achieve this, the lookup
Vindex types have an `ignore_nulls` option, e.g. for the vschema example above:

```
{
    "ks1": {
        "sharded": true,
        "vindexes": {
            "xxhash": {
                "type": "xxhash"
            },
            "lookup": {
                "type": "consistent_lookup_unique",
                "params": {
                    "table": "lookup1",
                    "from": "c2",
                    "to": "keyspace_id",
                    "ignore_nulls": "true"
                },
                "owner": "t1"
            }
        },
        "tables": {
            "t1": {
                "column_vindexes": [
                    {
                        "column": "c1",
                        "name": "xxhash"
                    },
                    {
                        "column": "c2",
                        "name": "lookup"
                    }
                ]
            },
            "lookup1": {
                "column_vindexes": [
                    {
                        "column": "c2",
                        "name": "xxhash"
                    }
                ]
            }
        }
    }
}
```

If we use this vschema, and attempt the same insert, we get:

```
$ vtexplain -schema-file schema.sql -vschema-file vschema.json -shards 8 -sql 'insert into t1 (c1, c2) values (1, NULL);'
----------------------------------------------------------------------
insert into t1 (c1, c2) values (1, NULL)

1 ks1/c0-e0: insert into t1(c1, c2) values (1, null)
```

i.e. the value is inserted into the sharded table (`t1`) based on the primary
Vindex column (`c1`); and the correponding insert to the lookup table (`lookup1`)
is skipped, because of the `ignore_nulls` option.

If a `SELECT` is made from `t1` passing no value for the primary Vindex column
(`c1`), and with a `NULL` value for the lookup Vindex column (`c2`), Vitess will
do the correct thing by:
  * first looking up the row corresponding to a `c2` value of `NULL` in the
  lookup table (it will **not** find a match)
  * then scattering the `SELECT` across all the shards

Unfortunately, we cannot illustrate this with vtexplain, since it cannot keep
lookup table state.  Also note that, in theory, an additional optimiztion is
possible for Vitess to just directly scatter a `SELECT` such as the above one,
without doing the lookup in the lookup table. Vitess does not do this today.
