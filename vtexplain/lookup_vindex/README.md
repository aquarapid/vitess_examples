Quick example of how to use vtexplain with a consistent lookup vindex

In this case, both the owner table for the lookup vindex and the
lookup vindex backing table are both sharded into the same keyspace.
vtexplain does not handle these being in separate keyspaces well.
