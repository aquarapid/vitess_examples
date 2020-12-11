Quick example of how to use vtexplain with a consistent lookup vindex

In this case, both the owner table for the lookup vindex and the
lookup vindex backing table are both sharded into the same keyspace.
vtexplain requires some more subtle syntax when you have these tables
in separate keyspaces, see ../lookup_vindex_unsharded/ for an
example.


Output when running this:

```sh
$ ./lookup_test.sh 
+ vtexplain -schema-file ./schema.sql -vschema-file ./vschema.json -shards 4 -sql 'select * from t1 where c1 = 9;'
----------------------------------------------------------------------
select * from t1 where c1 = 9

1 ks1/40-80: select * from t1 where c1 = 9 limit 10001

----------------------------------------------------------------------
+ vtexplain -schema-file ./schema.sql -vschema-file ./vschema.json -shards 4 -sql 'select * from c2_lookup where c2 = 243;'
----------------------------------------------------------------------
select * from c2_lookup where c2 = 243

1 ks1/c0-: select * from c2_lookup where c2 = 243 limit 10001

----------------------------------------------------------------------
+ vtexplain -schema-file ./schema.sql -vschema-file ./vschema.json -shards 4 -sql 'insert into t1(c1, c2) values (9,243);'
----------------------------------------------------------------------
insert into t1(c1, c2) values (9,243)

1 ks1/c0-: begin
1 ks1/c0-: insert into c2_lookup(c2, keyspace_id) values (243, 'i+��u+X')
2 ks1/40-80: begin
2 ks1/40-80: insert into t1(c1, c2) values (9, 243)
3 ks1/c0-: commit
4 ks1/40-80: commit

----------------------------------------------------------------------
```

