Quick example of how to use vtexplain with a consistent lookup vindex

In this case, both the owner table for the lookup vindex and the
lookup vindex backing table are in separate namespaces, with the
lookup vindex being in an unsharded namespace.  This is not
typically good practice, but this exposed a bug in vtexplain, i.e that
it cannot handle a mix of sharded and unsharded keyspaces.

This example assumes the fixed vtexplain.


Output when running this:

```sh
$ ./lookup_test.sh 
+ vtexplain -schema-file ./schema.sql -vschema-file ./vschema.json -shards 4 -sql 'select * from t1 where c1 = 9;'
----------------------------------------------------------------------
select * from t1 where c1 = 9

1 ks2/40-80: select * from t1 where c1 = 9 limit 10001

----------------------------------------------------------------------
+ vtexplain -schema-file ./schema.sql -vschema-file ./vschema.json -shards 4 -sql 'select * from c2_lookup where c2 = 243;'
----------------------------------------------------------------------
select * from c2_lookup where c2 = 243

1 ks1/-: select * from c2_lookup where c2 = 243 limit 10001

----------------------------------------------------------------------
+ vtexplain -schema-file ./schema.sql -vschema-file ./vschema.json -shards 4 -sql 'insert into t1(c1, c2) values (9,243);'
----------------------------------------------------------------------
insert into t1(c1, c2) values (9,243)

1 ks1/-: begin
1 ks1/-: select c2 from c2_lookup where c2 = 243 and keyspace_id = 'i+��u+X' limit 10001
2 ks2/40-80: begin
2 ks2/40-80: insert into t1(c1, c2) values (9, 243)
3 ks1/-: commit
4 ks2/40-80: commit

----------------------------------------------------------------------
```

