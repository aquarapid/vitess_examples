package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"strings"

	"vitess.io/vitess/go/vt/grpcclient"
	"vitess.io/vitess/go/vt/vtctl/grpcclientcommon"

	vtctldatapb "vitess.io/vitess/go/vt/proto/vtctldata"
	vtctlservicepb "vitess.io/vitess/go/vt/proto/vtctlservice"
)

func main() {
	ctx := context.Background()

	vtctld := flag.String("vtctld", "localhost:15999", "vtctld grpc host:port")
	args := flag.String("args", "", "argument(s) to send with command and arguments to vtctld")
	flag.Parse()
	if *vtctld == "" || *args == "" {
		fmt.Printf("Sample usage: go run grpc_vtctld_client.go -vtctld=localhost:15999 -args='ListAllTablets'\n")
		return
	}

	fmt.Printf("vtctld connecting to: %v\n\n", *vtctld)

	opt, err := grpcclientcommon.SecureDialOption()
	if err != nil {
		log.Fatal(err)
		return
	}
	conn, err := grpcclient.Dial(*vtctld, grpcclient.FailFast(false), opt)
	if err != nil {
		log.Fatal(err)
		return
	}
	defer conn.Close()

	client := vtctlservicepb.NewVtctlClient(conn)

	query := &vtctldatapb.ExecuteVtctlCommandRequest{
		Args:          strings.Split(*args, " "),
		ActionTimeout: int64(30000000000), // 30 secs in nanos
	}

	stream, err := client.ExecuteVtctlCommand(ctx, query)

	for {
		le, err := stream.Recv()
		if err != nil {
			break
		}
		fmt.Printf("%s", le.Event.Value)
	}
}
