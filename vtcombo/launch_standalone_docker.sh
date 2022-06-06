#!/bin/bash
vtcombo \
  -logtostderr=true \
  -proto_topo "$(cat /tmp/vitess/topo)" \
  -schema_dir /tmp/vitess/schema \
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