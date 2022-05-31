# `vtcombo`

An example of how to launch `vtcombo` against a local MySQL server instance.
We will cover two scenarios:
  * Running the `vtcombo` binary directly
  * Using a docker container to run `vtcombo`
We assume that the MySQL server is already running locally (`127.0.0.1`) on
port `3306`, the instance has been setup with the super/dba user `root` and
the password `password`.

Note that the `vtcombo` invocations presented here mainly rely on Vitess
component default parameters, and you may want to customize them depending
on which Vitess features you are using, and which parameters you have
customized.


## Running `vtcombo` directly

If you have the `vtcombo` binary available locally to run directly, you can
create a wrapper script to execute it, adding whatever `vtgate` and `vttablet`
flags you want to customize.  A reasonable example may be:

```
vtcombo \
  -logtostderr=true \
  -proto_topo "$(cat `pwd`/topo)" \
  -schema_dir `pwd`/schema \
  -mysql_server_port 55306 \
  -mysql_server_bind_address 127.0.0.1 \
  -mysql_auth_server_impl none \
  -db_host localhost \
  -db_port 3306 \
  -db_app_user root \
  -db_app_password password \
  -db_allprivs_user root \
  -db_allprivs_password password \
  -db_appdebug_user root \
  -db_appdebug_password password \
  -db_dba_user root \
  -db_dba_password password \
  -db_repl_user root \
  -db_repl_password password \
  -dbddl_plugin vttest \
  -health_check_interval 5s \
  -queryserver-config-query-timeout 30 \
  -queryserver-config-transaction-timeout 30 \
  -enable_system_settings=true \
  -port 55000 \
  -grpc_port 55001 \
  -service_map 'grpc-vtgateservice,grpc-vtctl,grpc-vtctld' \
  -vschema_ddl_authorized_users='%'
```

Note that if you are still running against a MySQL 5.7 server, you may want to
add an option to customize the MySQL version, e.g.

```
  -mysql_server_version "5.7.30-vitess"
```

In addition, you will need files in the local directory `schema/` to contain
the vschema (JSON) for each keyspace you want to create.
See the files in the `schema/` directory in this repo for an example.
We also include the schema (SQL) for each table in there.  `vtcombo` does not
execute these automatically, you will still need to apply these against
the appropriate keyspaces (schemas) via a MySQL client.

See the `launch_standalone.sh` script for the above.


## Running `vtcombo` using docker

See the `launch_docker.sh` file.  Note that this uses docker "host" networking
to allow the container to connect to a MySQL instance you have running
locally outside of the Vitess docker container.  It will also expose all the 
`vtcombo` ports on the local instance (`127.0.0.1`).  Accordingly, you will not
be able to run multiple copies of the container at the same time without
making some changes (e.g. each container will have to listen on different
ports).

Note that to launch `vtcombo` inside the container, the script
`launch_standalone_docker.sh` is used, if you want to make changes to
the `vtcombo` parameters and you are using docker, you will have to edit
that script.


## General notes

`vtcombo` combines multiple Vitess components into a single process:
  * A topo (topology) server.  In this case a "fake" one, not a real one
  you would use for production (e.g. `etcd`).  This "topology" server
  is populated by the information you provide in the `topo` file (which
  is provided to `vtcombo` via the `-proto_topo` parameter).
  * `vtgate`
  * `vttablet`
  * `vtctld`
It exposes the `vtctld` web UI on port `55000` in the above example.
It does not provide/expose the `vtgate` or `vttablet` web UIs.

All the gRPC functions for both `vtctld` and `vtgate` are exposed on the
single gRPC port of `55001` in the above example.  You can add exposing
the `vttablet` gRPC functions by editing the `-service-map` parameter.

Lastly, once `vtcombo` is up and running successfully, you should be able
to access Vitess (`vtgate`) via port `55306` on `127.0.0.1` using a
MySQL client.  Note that the authentication implementation is configured
as `none`;  meaning you can connect with any username/password combination
and will be let in.
