# PlanetScaleDB Operator with external Aurora tablets

Note that this example uses the proprietary PlanetScaleDB Operator from
PlanetScale (PSOP in the rest of this document), and will not work as-is
with the OSS PlanetScale Operator for Kubernetes.

## Pre-flight steps

Preparation steps that we assume you have completed:
  * Installed the PSOP operator (and Prometheus operator, if necessary)
  * Installed the necessary `ps-gcr-credentials` to access the PlanetScale
  private container repo;  or mirrored the images appropriately to your
  own container registry, and modified the various repo references
  appropriately.
  * Installed the PSOP operator (and first, Prometheus operator, if necessary)

## Setup

Once you have these steps completed, you can inspect and edit the
`external_multizone.yaml` file included in this repo appropriately,
modifying as necessary:
  * The cluster, keyspace, and database names
  * The hostnames for your Aurora instance RW and RO endpoints
  * The Aurora master and replica usernames and passwords
  * Adjust the number of replicas for your use-case
  * Adjust the cell and zone configuration and names to fit your use-case.
  In the example, we were running the PSOP on GKE in us-central1, in zones
  us-central1-a, us-central1-c and us-central1-f.
  * Adjust the memory and CPU sizing of the tablets (vttablet) and gateway
  (vtgate) instances appropriately for your workload.
  * Review the security options for your install.  For example, you may
  want to provision the sensitive k8s secrets included in the YAML separately
  via your infrastructure-as-code or gitops system you use with your k8s
  cluster.  You will also want to use hashed passwords for the
  vtgate credentials.  You will also want to select secure passwords.

Next, you can provision your cluster.  In our example, we are running
the PSOP in a namespace called `planetscale`, not the default namespace:

  * First, lets see what is running now:
  ```
$ kubectl -n planetscale get pod 
NAME                                     READY   STATUS    RESTARTS   AGE
planetscale-operator2-75ccd9444d-l6f2q   1/1     Running   0          65m
prometheus-operator-5674798fd9-cz67p     1/1     Running   0          66m
  ```
  * Now, lets apply our YAML:
  ```
$ kubectl apply -f external_multizone.yaml -n planetscale
planetscalecluster.planetscale.com/example-external-multizone created
secret/example-cluster-config created
  ```
  * Give it a few minutes, and with some luck, everything should come up fine:
  ```
$ kubectl -n planetscale get pod 
NAME                                                                  READY   STATUS    RESTARTS   AGE
example-external-multizone-etcd-1a7acab3-1                            1/1     Running   0          63s
example-external-multizone-etcd-1a7acab3-2                            1/1     Running   0          63s
example-external-multizone-etcd-1a7acab3-3                            1/1     Running   0          62s
example-external-multizone-grafana-5a257976-bc54f56c6-rz4wh           1/1     Running   0          63s
example-external-multizone-uscentral1a-vtctld-1a7eeadd-757gbqkx       1/1     Running   2          62s
example-external-multizone-uscentral1a-vtgate-5b063d95-86ccmrkr       0/1     Running   2          63s
example-external-multizone-uscentral1a-vtgate-5b063d95-86cnr7bt       1/1     Running   2          63s
example-external-multizone-uscentral1c-vtctld-0baa32d2-6fdwm27s       1/1     Running   2          63s
example-external-multizone-uscentral1c-vtgate-12cbfec4-69dshf54       1/1     Running   2          63s
example-external-multizone-uscentral1c-vtgate-12cbfec4-69dxnwwc       1/1     Running   2          63s
example-external-multizone-uscentral1f-vtctld-f81996f2-5c6r2q9v       1/1     Running   2          63s
example-external-multizone-uscentral1f-vtgate-f18ddec0-7dbk92t7       1/1     Running   2          63s
example-external-multizone-uscentral1f-vtgate-f18ddec0-7dbztcjn       1/1     Running   2          63s
example-external-multizone-vttablet-uscentral1a-0228574624-5fa97313   0/1     Running   2          63s
example-external-multizone-vttablet-uscentral1a-0693957838-4e30bc4e   0/1     Running   2          62s
example-external-multizone-vttablet-uscentral1c-0624489050-99b801b5   1/1     Running   2          63s
example-external-multizone-vttablet-uscentral1c-3044952688-761afb03   0/1     Running   2          62s
example-external-multizone-vttablet-uscentral1f-1117475966-012cc131   1/1     Running   2          63s
example-external-multizone-vttablet-uscentral1f-2162225014-6dac6b4a   0/1     Running   2          62s
planetscale-operator2-75ccd9444d-l6f2q                                1/1     Running   0          67m
prometheus-operator-5674798fd9-cz67p                                  1/1     Running   0          68m
  ```
  * Let's port-forward to one of the vtctld instances:
  ```
$ kubectl -n planetscale port-forward example-external-multizone-uscentral1a-vtctld-1a7eeadd-757gbqkx 15999:15999
Forwarding from 127.0.0.1:15999 -> 15999
Forwarding from [::1]:15999 -> 15999
  ```
  * Now, let's look at the instances running (the names are edited to obscure
  the upstream aurora instance names):
  ```
$ vtctlclient -server localhost:15999 ListAllTablets
uscentral1a-0228574624 keyspace1 - master 10.0.1.39:15000 aurora-rw-endpoint.rds.amazonaws.com:3306 [] 2021-02-17T04:37:45Z
uscentral1a-0693957838 keyspace1 - replica 10.0.1.44:15000 aurora-ro-endpoint.rds.amazonaws.com:3306 [] <null>
uscentral1c-0624489050 keyspace1 - spare 10.0.0.23:15000 aurora-rw-endpoint.rds.amazonaws.com:3306 [] <null>
uscentral1c-3044952688 keyspace1 - replica 10.0.0.26:15000 aurora-ro-endpoint.rds.amazonaws.com:3306 [] <null>
uscentral1f-1117475966 keyspace1 - spare 10.0.2.28:15000 aurora-rw-endpoint.rds.amazonaws.com:3306 [] <null>
uscentral1f-2162225014 keyspace1 - replica 10.0.2.29:15000 aurora-ro-endpoint.rds.amazonaws.com:3306 [] <null>
  ```
  * Inspecting this output, we see we have one master tablet , two spare
  tablets and 3 replica tablets. This is because we configured one external
  master tablet per zone (3 external master tablets total) and one external
  replica tablet per zone (3 external replica tablets total).  But because
  of the way Vitess works, we can only have one master for a shard at a
  time.  As a result only one of the external master tablets has a
  `tablet_type` of `MASTER` currently.  The other two external master tablets
  are created, but run as `tablet_type` of `SPARE`.  This means that no
  traffic will be routed to them. However, all three replica tablets
  are up, and available for sending read-only traffic to, using the usual
  Vitess routing (i.e. `@replica` qualified schema/database name).
  * When you want to redirect traffic to one of the `SPARE` tablets,
  you can just use the Vitess `TabletExternallyReparented` command. Any
  in-flight queries will complete, and new queries will be sent via the
  new `MASTER` tablet. Let's see this in action:
  ```
$ vtctlclient -server localhost:15999 TabletExternallyReparented uscentral1c-0624489050
  ```
  * This makes the new `MASTER` tablet the `SPARE` instance in the
  uscentral1c cell.  The current `MASTER` tablet instance will be demoted
  to a `SPARE` tablet_type.  Let's check that this is what has happened:
  ```
$ vtctlclient -server localhost:15999 ListAllTablets
uscentral1a-0228574624 keyspace1 - spare 10.0.1.39:15000 aurora-rw-endpoint.rds.amazonaws.com:3306 [] <null>
uscentral1a-0693957838 keyspace1 - replica 10.0.1.44:15000 aurora-ro-endpoint.rds.amazonaws.com:3306 [] <null>
uscentral1c-0624489050 keyspace1 - master 10.0.0.23:15000 aurora-rw-endpoint.rds.amazonaws.com:3306 [] 2021-02-17T04:47:50Z
uscentral1c-3044952688 keyspace1 - replica 10.0.0.26:15000 aurora-ro-endpoint.rds.amazonaws.com:3306 [] <null>
uscentral1f-1117475966 keyspace1 - spare 10.0.2.28:15000 aurora-rw-endpoint.rds.amazonaws.com:3306 [] <null>
uscentral1f-2162225014 keyspace1 - replica 10.0.2.29:15000 aurora-ro-endpoint.rds.amazonaws.com:3306 [] <null>
  ```
  * Indeed!
