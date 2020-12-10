#!/bin/sh
set -x

vtexplain -schema-file ./schema.sql -vschema-file ./vschema.json -shards 4 -sql 'select * from t1 where c1 = 9;'
vtexplain -schema-file ./schema.sql -vschema-file ./vschema.json -shards 4 -sql 'select * from c2_lookup where c2 = 243;'
vtexplain -schema-file ./schema.sql -vschema-file ./vschema.json -shards 4 -sql 'insert into t1(c1, c2) values (9,243);'
