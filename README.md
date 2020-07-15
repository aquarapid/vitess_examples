Some Vitess example code.

[vstream_test_client.go](vstream_test_client.go) - Commandline client to 
illustrate the use of the Vitess vStream API, and how vGTIDs work, 
including across multiple shards.

[vstream_test_client.md](vstream_test_client.md) - Walkthrough on how to use 
[vstream_test_client.go](vstream_test_client.go) to explore the workings of 
vStream and VGTIDs to stream changes from a Vitess database/keyspace.

[vstream_test_full.md](vstream_test_full.md) - Walkthrough on how to use 
[vstream_test_client.go](vstream_test_client.go) to see how vStream can be 
used to stream the full content **and** changes from a Vitess 
database/keyspace.
