# Vitess Operator RDS walkthrough

This is a quick walkthrough on standing up the Vitess Operator for Kubernetes
(or the proprietary analog, PlanetScaleDB for Kubernetes) in front of an
external MySQL installation.

In our example, we will be running the external databases (master and replica)
in AWS RDS, and the operator in a Kubernetes cluster in GKE, but this should
apply in broad strokes to any external MySQL installation and
standards-conformant Kubernetes install.

## Kubernetes operator setup

We will assume you already have the Kubernetes cluster created, and have
`kubectl` setup to run against it, and have the necessary K8s permissions
to install the operator.

We will make the simplifying assumption that you are installing the Operator
in the default k8s namespace.  Also, we are using a single-zone GKE cluster,
in the `us-central1-a` region, simplifying the need for having multiple
storage classes.

  * In our example, we have a 3 node cluster:

  ```
$ kubectl get node
NAME                                                STATUS   ROLES    AGE   VERSION
gke-jacques-uscentral1-default-pool-6025632a-4wr3   Ready    <none>   87m   v1.16.13-gke.401
gke-jacques-uscentral1-default-pool-6025632a-jn8h   Ready    <none>   87m   v1.16.13-gke.401
gke-jacques-uscentral1-default-pool-6025632a-rpsx   Ready    <none>   87m   v1.16.13-gke.401
  ```

  * As a first step, we create the necessary storage class:

  ```
$ cat gke_sc_single_zone.yaml
apiVersion: v1
items:
- allowVolumeExpansion: true
  apiVersion: storage.k8s.io/v1
  kind: StorageClass
  metadata:
    name: uscentral1a-ssd
  parameters:
    type: pd-ssd
    zone: us-central1-a
  provisioner: kubernetes.io/gce-pd
  reclaimPolicy: Delete
  volumeBindingMode: Immediate
kind: List

$ kubectl apply -f gke_sc_single_zone.yaml -n planetscale
storageclass.storage.k8s.io/uscentral1a-ssd created
  ```

  * Now, we install the Vitess operator:

  ```
$ kubectl apply -f operator.yaml
customresourcedefinition.apiextensions.k8s.io/etcdlockservers.planetscale.com created
customresourcedefinition.apiextensions.k8s.io/vitessbackups.planetscale.com created
customresourcedefinition.apiextensions.k8s.io/vitessbackupstorages.planetscale.com created
customresourcedefinition.apiextensions.k8s.io/vitesscells.planetscale.com created
customresourcedefinition.apiextensions.k8s.io/vitessclusters.planetscale.com created
customresourcedefinition.apiextensions.k8s.io/vitesskeyspaces.planetscale.com created
customresourcedefinition.apiextensions.k8s.io/vitessshards.planetscale.com created
serviceaccount/vitess-operator created
role.rbac.authorization.k8s.io/vitess-operator created
rolebinding.rbac.authorization.k8s.io/vitess-operator created
priorityclass.scheduling.k8s.io/vitess created
priorityclass.scheduling.k8s.io/vitess-operator-control-plane created
deployment.apps/vitess-operator created
  ```

  * Validate the operator is running:

  ```
$ kubectl get pods
NAME                               READY   STATUS    RESTARTS   AGE
vitess-operator-58d9c4967b-gdjc9   1/1     Running   0          37s
  ```


## RDS setup

We will assume that we have two RDS instances running:

* A master instance, called `rds-master` in our example.  Since we are running
our K8s cluster outside AWS, we have enabled internet access for this cluster,
and the DNS endpoint for this instance is: `rds-master.cc8e8tvt4276.us-west-2.rds.amazonaws.com`
* A read replica instance of `rds-master` called `rds-replica`. Again, we have
enabled internet access for this instance, and the DNS endpoint for it is:
`rds-replica.cc8e8tvt4276.us-west-2.rds.amazonaws.com`.

For RDS we have created an admin user called `rds_master` with
the password `passwords!!`.  This username/password combination
are of course valid for both instances.


## Review CRD configuration for Vitess

The Vitess operator takes CRD configuration via YAML input to create the
necessary components to run in front of the RDS instances.  We have provided
an example of this in the `exampledb.yaml` file in this directory.  Please
review at least the following parts of this configuration:

* DNS names for your RDS instances (search for `rds-master` and `rds-replica`)
* Usernames/passwords for your RDS instances (search for `rds_master`)
* Username/password for the Vitess instance MySQL protocol access (vtgate).
We have made that username `admin` and `passwords!!` in the example.
* Sizing changes for the RDS Vitess components, if necessary.  We have kept
them small to run even in tiny Kubernetes clusters.
* Note that we have provided default TLS certificates for vtgate, that may
not be trusted by your clients if you want to use TLS to speak to `vtgate`.
* We have also embedded the whole of the AWS RDS certificate bundle, which
should enable this configuration to talk to any external RDS instance.
* In this example, the database front-ended by Vitess is called `main`
on the external database instances;  and the keyspace we stand up in
front if it is also called `main`.  You should adjust this if necessary.
Also ensure that this database already exists in the external database.
In our example this database is empty, but it does not need to be.
* You should also create the Vitess "sidecar" database in the external
database instance. It should be called: `_vt`, e.g. when connected to
the master RDS instance:

```
mysql> create database _vt;
Query OK, 1 row affected (0.04 sec)
```


Now, you can apply the CRD yaml to the operator:

```
$ kubectl apply -f exampledb.yaml
vitesscluster.planetscale.com/example-external created
secret/example-cluster-config created
```

Now, validate the Vitess components are starting successfully (it may take a minute
or two for all the pods to transition to `Running` and all the pod containers
to be `READY`):

```
$ kubectl get pods
NAME                                                            READY   STATUS    RESTARTS   AGE
example-external-etcd-5bea2393-1                                1/1     Running   0          93s
example-external-etcd-5bea2393-2                                1/1     Running   0          93s
example-external-etcd-5bea2393-3                                1/1     Running   0          93s
example-external-uscentral1a-vtctld-27483830-5c889979d6-t72r7   1/1     Running   2          93s
example-external-uscentral1a-vtgate-b8f909fc-5b74cd9d88-k6wwl   1/1     Running   2          93s
example-external-uscentral1a-vtgate-b8f909fc-5b74cd9d88-vm8wt   1/1     Running   2          93s
example-external-vttablet-uscentral1a-1507852424-fd25d69b       1/1     Running   2          94s
example-external-vttablet-uscentral1a-4041589429-6af95183       1/1     Running   2          94s
vitess-operator-58d9c4967b-7md6z                                1/1     Running   0          23m
```

That seems good!  Now, we need to expose the vtgate service, so we can
get to it from outside the Kubernetes cluster:

```
$ kubectl expose deployment $(kubectl get deployment --selector="planetscale.com/component=vtgate" -o=jsonpath="{.items..metadata.name}") --type=LoadBalancer --name=vtgate-svc --port 3306 --target-port 3306
service/vtgate-svc exposed
$ kubectl get service vtgate-svc
NAME         TYPE           CLUSTER-IP      EXTERNAL-IP   PORT(S)          AGE
vtgate-svc   LoadBalancer   10.11.247.115   <pending>     3306:31188/TCP   37s
```

We may now have to wait a minute or two until the external cloud load balancer
assigns an `EXTERNAL-IP` to this service.  A minute later:

```
$ kubectl get service vtgate-svc
NAME         TYPE           CLUSTER-IP      EXTERNAL-IP      PORT(S)          AGE
vtgate-svc   LoadBalancer   10.11.247.115   34.122.110.122   3306:31188/TCP   93s
```

Now, we can connect to the vtgate instances via the external load balancer
IP, e.g.:

```
$ mysql -u admin -h 34.122.110.122 -p
Enter password: 
Welcome to the MySQL monitor.  Commands end with ; or \g.
Your MySQL connection id is 2
Server version: 5.7.9-Vitess

Copyright (c) 2009-2020 Percona LLC and/or its affiliates
Copyright (c) 2000, 2020, Oracle and/or its affiliates. All rights reserved.

Oracle is a registered trademark of Oracle Corporation and/or its
affiliates. Other names may be trademarks of their respective
owners.

Type 'help;' or '\h' for help. Type '\c' to clear the current input statement.

mysql>
```

Note the `Server version` reported is set by Vitess, and is unrelated to
the MySQL version of the underlying external MySQL instances. The version
reported by Vitess can be modified by adding a `vtgate` option to your
configuration. In this specific example the MySQL version of the underlying
RDS instances happens to be MySQL 8.0.20.

Now, we take a look at the default keyspace/database setup.  As per the CRD
yaml we provided to the operator, we created a single unsharded keyspace called
`main`.  We can also use some of the Vitess meta-commands available via the
MySQL protocol to see the configuration of the `vttablet` instances:

```
mysql> show databases;
+-----------+
| Databases |
+-----------+
| main      |
+-----------+
1 row in set (0.06 sec)

mysql> show vitess_tablets;
+-------------+----------+-------+------------+---------+------------------------+----------+
| Cell        | Keyspace | Shard | TabletType | State   | Alias                  | Hostname |
+-------------+----------+-------+------------+---------+------------------------+----------+
| uscentral1a | main     | -     | MASTER     | SERVING | uscentral1a-1507852424 | 10.8.1.7 |
| uscentral1a | main     | -     | REPLICA    | SERVING | uscentral1a-4041589429 | 10.8.1.5 |
+-------------+----------+-------+------------+---------+------------------------+----------+
2 rows in set (0.06 sec)
```


We can now go ahead and create tables, insert data, etc.:

```
mysql> use main
Database changed

mysql> create table t1 (c1 int, primary key (c1));
Query OK, 0 rows affected (0.14 sec)

mysql> insert into t1 (c1) values (11);
Query OK, 1 row affected (0.05 sec)

mysql> select * from t1;
+----+
| c1 |
+----+
| 11 |
+----+
1 row in set (0.04 sec)
```



## Options used by vttablet against external database

We can also get some insight into the default vttablet options used by the
operator against external MySQL instances by inspecting some of the
vttablet pods, e.g., against the master vttablet pod:

```
Name:                 example-external-vttablet-uscentral1a-1507852424-fd25d69b
Namespace:            default
Priority:             1000
Priority Class Name:  vitess
Node:                 gke-jacques-uscentral1-default-pool-6025632a-rpsx/10.128.0.77
Start Time:           Tue, 24 Nov 2020 16:56:21 -0800
Labels:               planetscale.com/cell=uscentral1a
                      planetscale.com/cluster=example-external
                      planetscale.com/component=vttablet
                      planetscale.com/keyspace=main
                      planetscale.com/shard=x-x
                      planetscale.com/tablet-index=1
                      planetscale.com/tablet-type=externalmaster
                      planetscale.com/tablet-uid=1507852424
Annotations:          drain.planetscale.com/supported: ensure that the tablet is not a master
                      kubernetes.io/limit-ranger: LimitRanger plugin set: cpu request for init container init-vt-root
                      planetscale.com/annotations-keys-hash: a9847536b7e2f45275c78fbd3eac6e8f
                      planetscale.com/containers-hash: 50673946d7d6690744c310d6c06c503c
                      planetscale.com/init-containers-hash: 94d7281d9fe202873cec1a7738dc3b69
                      planetscale.com/labels-keys-hash: d41d8cd98f00b204e9800998ecf8427e
                      planetscale.com/observed-shard-generation: 1
Status:               Running
IP:                   10.8.1.7
IPs:
  IP:           10.8.1.7
Controlled By:  VitessShard/example-external-main-x-x-4679a438
Init Containers:
.
.
.
.
Containers:
  vttablet:
    Container ID:  docker://347bbd51e731f65f272da1c32ca800f616aacc7c4cf1923dab2fe8cffb478317
    Image:         us.gcr.io/planetscale-vitess/lite:2020-03-19.771aa75d
    Image ID:      docker-pullable://us.gcr.io/planetscale-vitess/lite@sha256:898b286ac8c779c9cfde68a5340c6a3b9a511f8458b740d0784ceeff5da92dd9
    Ports:         15000/TCP, 15999/TCP
    Host Ports:    0/TCP, 0/TCP
    Command:
      /vt/bin/vttablet
    Args:
      --binlog_use_v3_resharding_mode=true
      --client-found-rows-pool-size=300
      --db-credentials-file=/vt/secrets/external-datastore-credentials/master_creds.json
      --db_allprivs_user=rds_master
      --db_app_user=rds_master
      --db_appdebug_user=rds_master
      --db_charset=utf8mb4
      --db_dba_user=rds_master
      --db_filtered_user=rds_master
      --db_flags=2048
      --db_host=rds-master.cc8e8tvt4276.us-west-2.rds.amazonaws.com
      --db_port=3306
      --db_repl_user=rds_master
      --db_ssl_ca=/vt/secrets/external-datastore-ca-cert/master_ca.pem
      --disable_active_reparents=true
      --enable_hot_row_protection=true
      --enable_replication_reporter=true
      --enforce_strict_trans_tables=false
      --grpc_max_message_size=67108864
      --grpc_port=15999
      --health_check_interval=5s
      --init_db_name_override=main
      --init_keyspace=main
      --init_shard=-
      --init_tablet_type=spare
      --logtostderr=true
      --port=15000
      --queryserver-config-max-result-size=100000
      --queryserver-config-pool-size=96
      --queryserver-config-query-cache-size=0
      --queryserver-config-query-timeout=900
      --queryserver-config-schema-reload-time=60
      --queryserver-config-stream-pool-size=96
      --queryserver-config-transaction-cap=300
      --queryserver-config-transaction-timeout=300
      --restore_from_backup=false
      --service_map=grpc-queryservice,grpc-tabletmanager,grpc-updatestream
      --tablet-path=uscentral1a-1507852424
      --tablet_hostname=$(POD_IP)
      --topo_global_root=/vitess/example-external/global
      --topo_global_server_address=example-external-etcd-5bea2393-client:2379
      --topo_implementation=etcd2
      --vreplication_tablet_type=master
    State:          Running
      Started:      Tue, 24 Nov 2020 16:56:49 -0800
    Last State:     Terminated
      Reason:       Error
      Exit Code:    1
      Started:      Tue, 24 Nov 2020 16:56:30 -0800
      Finished:     Tue, 24 Nov 2020 16:56:35 -0800
    Ready:          True
    Restart Count:  2
    Limits:
      memory:  384Mi
    Requests:
      cpu:      250m
      memory:   256Mi
    Liveness:   http-get http://:web/debug/status delay=300s timeout=1s period=10s #success=1 #failure=30
    Readiness:  http-get http://:web/healthz delay=0s timeout=1s period=10s #success=1 #failure=3
    Environment:
      VTROOT:         /vt
      VTDATAROOT:     /vt/vtdataroot
      VT_MYSQL_ROOT:  /usr
      MYSQL_FLAVOR:   MySQL56
      EXTRA_MY_CNF:   /vt/config/mycnf/rbr.cnf:/vt/config/mycnf/log-error.cnf:/vt/config/mycnf/socket.cnf
      POD_IP:          (v1:status.podIP)
    Mounts:
      /var/run/secrets/kubernetes.io/serviceaccount from default-token-zbrp5 (ro)
      /vt/config from vt-root (ro,path="config")
      /vt/secrets/db-init-script from db-init-script-secret (ro)
      /vt/secrets/external-datastore-ca-cert from external-datastore-ca-cert-secret (ro)
      /vt/secrets/external-datastore-credentials from external-datastore-credentials-secret (ro)
      /vt/socket from vt-root (rw,path="socket")
      /vt/vtdataroot from vt-root (rw,path="vtdataroot")
Conditions:
  Type              Status
  Initialized       True
  Ready             True
  ContainersReady   True
  PodScheduled      True
Volumes:
  external-datastore-credentials-secret:
    Type:        Secret (a volume populated by a Secret)
    SecretName:  example-cluster-config
    Optional:    false
  external-datastore-ca-cert-secret:
    Type:        Secret (a volume populated by a Secret)
    SecretName:  example-cluster-config
    Optional:    false
  vt-root:
    Type:       EmptyDir (a temporary directory that shares a pod's lifetime)
    Medium:
    SizeLimit:  <unset>
  db-init-script-secret:
    Type:        Secret (a volume populated by a Secret)
    SecretName:  example-cluster-config
    Optional:    false
  default-token-zbrp5:
    Type:        Secret (a volume populated by a Secret)
    SecretName:  default-token-zbrp5
    Optional:    false
QoS Class:       Burstable
Node-Selectors:  <none>
Tolerations:     node.kubernetes.io/not-ready:NoExecute op=Exists for 300s
                 node.kubernetes.io/unreachable:NoExecute op=Exists for 300s
```
