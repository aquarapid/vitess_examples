Example on how to use gRPC with PlanetScale:

  * The gRPC protos are in the `proto` directory in this repo.
  * You will need to clone the (public) planetscale/vitess-types
    repo.
  * The `Authorization` header is in the format of your PlanetScale
    `username:password`, base64-encoded.
  * Reflection is not supported by the gRPC server, look in the 
    `proto/psdb.v1alpha1.proto` file for the interface.  You probably
    only need `StreamExecute` if you plan on doing ETL-y things.
  * `aws.connect.psdb.cloud` is the generic AWS endpoint (us-east-1) for
    PlanetScale, you may need to replace depending on where (and how) your
    PlanetScale database is deployed.

Example of executing `SELECT 1` via `grpcurl`:

```
$ grpcurl -H "authorization: Basic base64-encoded-username-colon-password" -d '{"query": "select 1"}' -proto psdb.v1alpha1.proto -import-path proto -import-path ../../vitess-types/src aws.connect.psdb.cloud:443 psdb.v1alpha1.Database/StreamExecute
```

