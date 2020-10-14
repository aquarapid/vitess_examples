# MoveTables reads/writes switchover:


## Rollback steps:

### **ONLY** if writes have not been switched yet:

  * Note that these do not affect the actual flow direction of vreplication,
    but only query routing by Vitess.

RDONLY:

```
$ vtctlclient -server vtctld:15999 SwitchReads -tablet_type=rdonly -reverse targetks.workflowname
```

REPLICA:

```
$ vtctlclient -server vtctld:15999 SwitchReads -tablet_type=replica -reverse targetks.workflowname
```


### Once writes have been switched:

  * When we did a `SwitchWrites` for the original `MoveTables` workflow, a
    reverse workflow, called `workflowname_reverse` (basically the original
    workflow name with a `_reverse` added) was created.  Writes to the target
    keyspace are now being vreplicated back to the original source keyspace.
    We can use the same procedure to switch reads and writes back to the
    source keyspace as we followed when switching them to the target keyspace.
    The only difference is the keyspace name used (the workflow is now hosted
    by the source keyspace);  and the new workflow name.

RDONLY:

```
$ vtctlclient -server vtctld:15999 SwitchReads -tablet_type=rdonly sourceks.workflowname_reverse
```

REPLICA:

```
$ vtctlclient -server vtctld:15999 SwitchReads -tablet_type=replica sourceks.workflowname_reverse
```


WRITES (all reads **must** already be switched back):

```
$ vtctlclient -server vtctld:15999 SwitchWrites sourceks.workflowname_reverse
```
