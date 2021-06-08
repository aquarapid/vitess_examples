# Ports and network interactions in Vitess

Many/most of these ports are fully configurable, but we are listing their
defaults or by convention (e.g. from the examples) defaults here. Your
environment may differ considerably, depending on your configuration options
for the various components:

  * Data path - e.g.:
    * Main query path:
      * application -> vtgate
        * port 3306 or 15306 (MySQL)
        * port 15999 (gRPC)
      * vtgate -> vttablet
        * port 16000 + vttablet UID (gRPC)
      * vttablet -> MySQL
        * local socket (if MySQL is local)
        * port 3306 (if MySQL is remote)
    * vttablet -> vttablet:  vreplication within or across shards
      * port 16000 + vttablet UID (gRPC)
    * MySQL -> MySQL:  within-shard replication
      * port 3306 (MySQL protocol)
  * Control or meta-data paths - e.g.:
    * vtctld -> vttablet
      * port 16000 + vttablet UID (gRPC)
    * vtctlclient -> vtctld
      * port 15999 (gRPC)
      * configured via `-server` option
    * vtgate -> topology server
      * Depends on topology server, e.g. for etcd typically port 2379
      * configured via `-topo_global_server_address`
    * vttablet -> topology server
      * Depends on topology server, e.g. for etcd typically port 2379
      * configured via `-topo_global_server_address`
    * vtctld -> topology server
      * Depends on topology server, e.g. for etcd typically port 2379
      * configured via `-topo_global_server_address`
    * administrator using web browser -> vtgate web UI
      * port 15001 (HTTP)
    * administrator using web browser -> vttablet web UI
      * port 15000 + vttablet UID (HTTP)
    * administrator using web browser -> vtctld web UI
      * port 15000 (HTTP)
    * Metrics scraper (e.g. Prometheus) -> vtgate web port
      * port 15001 (HTTP)
    * Metrics scraper (e.g. Prometheus) -> vttablet web port
      * port 15000 + vttablet UID (HTTP)
    * Metrics scraper (e.g. Prometheus) -> vtctld web port
      * port 15000 (HTTP)

