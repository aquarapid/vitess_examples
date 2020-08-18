Example procedure for backup/restore with the PlanetScaleDB Operator:

1. Setup the `spec.backup` section of your PlanetScaleCluster (PSC) CRD
   definition appropriately, e.g.:
   ```
   backup:
     locations:
     - gcs:
         bucket: my-bucket
         authSecret:
           name: secretname
           key: key.json
     engine: xtrabackup
     policy:
       backupIntervalHours: 24
       minRetentionHours: 72
       minRetentionCount: 1
   ```
   Note especially the `engine` definition;  without this you will get
   the `builtin` default engine, which does not do online backups.
   See the operator documentation for more details on the cloud-specific
   (GCS, S3, etc) backup location options.

   Also ensure that the storage bucket is empty before starting.  The
   Operator will populate backups from the contents of a bucket that
   has been previously used for the backups of another (or earlier
   version of) your Vitess Cluster.  You can, however, share the same
   bucket between different Vitess Clusters with different names.

   Lastly, note the backup policy section;  this allows the backup
   frequency and retention rules to be specified. The values provided above
   are also the default values if no values are specified.

1. Deploy the PSC via the operator.  After all the Vitess cluster pods
   come up (i.e. are in `Running` state), you can run a manual backup.
   To do this, setup port-forwarding via your vtctld pod, e.g.:

   `$ kubectl -n planetscale port-forward $(kubectl -n planetscale get pod | grep vtctld | head -1 | awk '{ print $1 }') 16999:15999`

1. In a different window, you should now be able to run `vtctlclient`
   against your cluster, against `localhost` port `16999`, e.g.:
   ```
   $ vtctlclient -server localhost:16999 ListAllTablets
   uscentral1a-0274179421 restore 80- master 10.32.0.184:15000 10.32.0.184:3306 [] 2020-08-12T04:39:06Z
   uscentral1a-1579720563 main 80- replica 10.32.0.183:15000 10.32.0.183:3306 [] <null>
   uscentral1a-3642477526 restore -80 master 10.32.0.186:15000 10.32.0.186:3306 [] 2020-08-12T04:39:06Z
   uscentral1a-3876690474 main -80 replica 10.32.0.182:15000 10.32.0.182:3306 [] <null>
   uscentral1c-2526979163 main 80- replica 10.32.2.148:15000 10.32.2.148:3306 [] <null>
   uscentral1c-3925422610 main -80 master 10.32.2.149:15000 10.32.2.149:3306 [] 2020-08-12T04:39:06Z
   uscentral1f-0805900210 main 80- master 10.32.1.111:15000 10.32.1.111:3306 [] 2020-08-12T04:38:50Z
   uscentral1f-2191363457 main -80 replica 10.32.1.112:15000 10.32.1.112:3306 [] <null>
   ```

   Note that in this configuration we have two namespaces:
     * `main`: sharded namespace we are going to take backups of.
     * `restore`:  sharded namespace we are going to restore to.

   We could also have started the cluster with just a `main` keyspace, and
   then added the `restore` keyspace later, using `ps-rollout` to reconcile
   and add the new namespace.

   One last thing that is important to note is the `databaseName`
   option in the CRD for the cluster for the `restore` keyspace, e.g.:

   ```
   - name: restore
     databaseName: vt_main
     partitionings:
     - equal:
     .
     .
   ```

   To allow a restore, the `databaseName` needs to match the name of
   the database when it was backed up.  In our case the keyspace being
   backed up is called `main`, and thus the default backing database
   in MySQL that is getting backed up is called `vt_main` (unless
   you overrode it).  Thus, for the `restore` keyspace, we need to
   specify the same database name.

1. Validate that there are only the new (initial) backups recorded for the
shards in the keyspaces:
   ```
   $ kubectl -n planetscale get vitessbackups
   NAME                                                     AGE
   example-main-80-x-20200812-043818-38875fa9-ae3cf855      49s
   example-main-x-80-20200812-043833-e198004a-f2e040fe      50s
   example-restore-80-x-20200812-043833-d46c6a64-cec703c3   49s
   example-restore-x-80-20200812-043835-86d27b64-a19a3004   49s
   ```

1. Insert some data into your cluster (so that there is something to back
   up).  You may need to apply a VSchema and SQL schema first, e.g.:
   ```
   $ vtctlclient -server localhost:16999 ApplyVSchema -vschema="$(cat ./vschema.json)" main
   $ vtctlclient -server localhost:16999 ApplySchema -sql="$(cat ./schema.sql)" main
   ```
   You may need to expose your vtgate instances to be able to connect to it.
   Consult the getting started guide. After you can connect, insert some data
   into your database.

1. Now, you can take a manual backup for each shard, e.g.:
   ```
   $ vtctlclient -server localhost:16999  BackupShard  main/-80
   $ vtctlclient -server localhost:16999  BackupShard  main/80-
   ```
   In my case, I ran these backups twice, adding some more data to the
   database between runs, so I can illustrate how to restore a backup
   older than the "most recent" one.
1. Now, you should be able to see the actual backups via the CRD actions
   against k8s, e.g.:
   ```
   $ kubectl -n planetscale get vitessbackups
   NAME                                                     AGE
   example-main-80-x-20200812-043818-38875fa9-ae3cf855      3m27s
   example-main-80-x-20200812-044113-5e289f73-076378c5      76s
   example-main-80-x-20200812-044154-5e289f73-fe4cc509      20s
   example-main-x-80-20200812-043833-e198004a-f2e040fe      3m28s
   example-main-x-80-20200812-044109-829d8d81-5d46a4f0      76s
   example-main-x-80-20200812-044148-e7119a2a-1cf1d4f0      19s
   example-restore-80-x-20200812-043833-d46c6a64-cec703c3   3m27s
   example-restore-x-80-20200812-043835-86d27b64-a19a3004   3m27s
   ```
   Note that the backups might not appear for an extended time if the
   shards being backed up are large;  and will take a minute or two, even if
   the shard is small.

1. Now, to restore a backup, there are two ways:
   * Delete your whole cluster, and re-create it.  If you are using the same
     shard configuration and pointing to the same (now non-empty) backup
     storage bucket, the newest completed backup for the shards will be
     restored to your cluster.  This will not be a PITR restore, since you
     will have no binary logs available to level the restored shards.
   * The alternate option is to create a new keyspace, and use it for
     restoring the backup from your original keyspace (called `main` in
     the above example).  We will follow this method.

1. In our example, we created a "restore" keyspace with 1 non-master tablet
   per shard.  We're going to delete the Vitess toplogy record for this
   keyspace, and then re-initialize it with "snapshot" options, so we
   can tell it to restore a specific backup:

   * First delete the keyspace information:

   ```
   $ vtctlclient -server localhost:16999 DeleteKeyspace -recursive restore
   ```

   * Next, run CreateKeyspace to recreate the keyspace, with the appropriate
     options:

   `$ vtctlclient -server localhost:16999 CreateKeyspace -keyspace_type=SNAPSHOT -base_keyspace=main -snapshot_time=2020-08-12T04:41:15+00:00 restore`

   This will enable the restore the older of the 2 manual backups, since the 
   snapshot time provided is after the first 2 backups, but before the next set
   of backups.
   For good measure, check the keyspace was created with the correct options,
   e.g.:

   ```
   $ vtctlclient -server localhost:16999  GetKeyspace restore
   {
     "sharding_column_name": "",
     "sharding_column_type": 0,
     "served_froms": [
     ],
     "keyspace_type": 1,
     "base_keyspace": "main",
     "snapshot_time": {
       "seconds": "1597207275",
       "nanoseconds": 0
     }
   }
   ```

   * Next, delete the pods that was started for the `restore` keyspace. To
     find the pod names for these, we can use the `ListAllTablets` output
     we had earlier, before we deleted the keyspace, e.g.:

   `$ kubectl -n planetscale get pod  | grep -e uscentral1a-0274179421 -e uscentral1a-3642477526`
   ```
   example-vttablet-uscentral1a-0274179421-446d19ea       3/3     Running   2     8m54s
   example-vttablet-uscentral1a-3642477526-9ef49ae4       3/3     Running   2     8m53s
   ```

     So, we're going to delete these two tablets:

   `$ kubectl -n planetscale delete pod example-vttablet-uscentral1a-0274179421-446d19ea example-vttablet-uscentral1a-3642477526-9ef49ae4`
   ```
   pod "example-vttablet-uscentral1a-0274179421-446d19ea" deleted
   pod "example-vttablet-uscentral1a-3642477526-9ef49ae4" deleted
   ```

   * The pods should be recreated, and initialized from the snapshot backup
     as specified in the `CreateKeyspace` options.  During the restore, the
     pods will have the tablet type of `restore`, as can be seen in the
     `ListAllTablets` output, e.g.:

   ```
   $ vtctlclient -server localhost:16999 ListAllTablets   
   uscentral1a-0274179421 restore 80- restore 10.32.0.191:15000 10.32.0.191:3306 [] <null>
   uscentral1a-1579720563 main 80- replica 10.32.0.183:15000 10.32.0.183:3306 [] <null>
   uscentral1a-3642477526 restore -80 restore 10.32.0.192:15000 10.32.0.192:3306 [] <null>
   uscentral1a-3876690474 main -80 replica 10.32.0.182:15000 10.32.0.182:3306 [] <null>
   uscentral1c-2526979163 main 80- replica 10.32.2.148:15000 10.32.2.148:3306 [] <null>
   uscentral1c-3925422610 main -80 master 10.32.2.149:15000 10.32.2.149:3306 [] 2020-08-12T04:39:06Z
   uscentral1f-0805900210 main 80- master 10.32.1.111:15000 10.32.1.111:3306 [] 2020-08-12T04:38:50Z
   uscentral1f-2191363457 main -80 replica 10.32.1.112:15000 10.32.1.112:3306 [] <null>
   ```

   * After the restore is complete, the tablet types should transition to
     `master`, e.g.:

   ```
   $ vtctlclient -server localhost:16999 ListAllTablets
   uscentral1a-0274179421 restore 80- master 10.32.0.191:15000 10.32.0.191:3306 [] 2020-08-12T04:48:47Z
   uscentral1a-1579720563 main 80- replica 10.32.0.183:15000 10.32.0.183:3306 [] <null>
   uscentral1a-3642477526 restore -80 master 10.32.0.192:15000 10.32.0.192:3306 [] 2020-08-12T04:48:53Z
   uscentral1a-3876690474 main -80 replica 10.32.0.182:15000 10.32.0.182:3306 [] <null>
   uscentral1c-2526979163 main 80- replica 10.32.2.148:15000 10.32.2.148:3306 [] <null>
   uscentral1c-3925422610 main -80 master 10.32.2.149:15000 10.32.2.149:3306 [] 2020-08-12T04:39:06Z
   uscentral1f-0805900210 main 80- master 10.32.1.111:15000 10.32.1.111:3306 [] 2020-08-12T04:38:50Z
   uscentral1f-2191363457 main -80 replica 10.32.1.112:15000 10.32.1.112:3306 [] <null>
   ```

   * You should now be able to access the restored data via the `restore`
     keyspace by connecting to vtgate:

```
   $ mysql -u user -p -h vtgate.ip.address -A
   Enter password:
   Welcome to the MySQL monitor.  Commands end with ; or \g.
   .
   .
   .
   
   mysql> show databases;
   +-----------+
   | Databases |
   +-----------+
   | main      |
   | restore   |
   +-----------+
   2 rows in set (0.05 sec)
   
   mysql> use restore;
   Database changed
   mysql> show tables;
   +-------------------+
   | Tables_in_vt_main |
   +-------------------+
   | users             |
   | users_name_idx    |
   +-------------------+
   2 rows in set (0.05 sec)
   
   mysql> select * from users;
   +---------+---------+
   | user_id | name    |
   +---------+---------+
   |     101 | Jacques |
   +---------+---------+
   1 row in set (0.06 sec)
   
   mysql> use main;
   Database changed
   mysql> select * from users;
   +---------+---------+
   | user_id | name    |
   +---------+---------+
   |     101 | Jacques |
   |     102 | Jacques |
   +---------+---------+
   2 rows in set (0.05 sec)
```

This therefore showed that the restored `restore` keyspace was from an
older backup than the most up-to-date data in the `main` keyspace.


