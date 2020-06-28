package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"time"

	"github.com/olekukonko/tablewriter"
	"vitess.io/vitess/go/sqltypes"
	binlogdatapb "vitess.io/vitess/go/vt/proto/binlogdata"
	querypb "vitess.io/vitess/go/vt/proto/query"
	topodatapb "vitess.io/vitess/go/vt/proto/topodata"
	_ "vitess.io/vitess/go/vt/vtctl/grpcvtctlclient"
	_ "vitess.io/vitess/go/vt/vtgate/grpcvtgateconn"
	"vitess.io/vitess/go/vt/vtgate/vtgateconn"
)

func main() {
	ctx := context.Background()

	vtgate := flag.String("vtgate", "localhost:15991", "vtgate grpc host:port")
	keyspace := flag.String("keyspace", "", "keyspace to filter on")
	tabletType := flag.String("tablet_type", "", "master/replica/rdonly")
	pos := flag.String("pos", "", "JSON structure describing vtgtids to start from")
	flag.Parse()
	if *vtgate == "" || *keyspace == "" || *tabletType == "" || *pos == "" {
		fmt.Printf("Sample usage: go run custom_vstream.go -vtgate=localhost:15991 -keyspace=commerce -tablet_type=master -pos='[ {\"shard\":\"x\", \"gtid\":\"xxxx\"} ]'\n")
		return
	}

	var shardgtids []*binlogdatapb.ShardGtid
	json.Unmarshal([]byte(*pos), &shardgtids)

	for _, shardgtid := range shardgtids {
		//fmt.Printf("debug shardgtid: %v\n", shardgtid)
		shardgtid.Keyspace = *keyspace
	}

	vgtid := &binlogdatapb.VGtid{
		ShardGtids: shardgtids,
	}

	tt := topodatapb.TabletType_RDONLY
	switch *tabletType {
	case "master":
		tt = topodatapb.TabletType_MASTER
	case "replica":
		tt = topodatapb.TabletType_REPLICA
	case "rdonly":
		tt = topodatapb.TabletType_RDONLY
	}
	fmt.Printf("vtgate connecting to: %v\nvgtid: %v\ntablet_type: %v\n\n", *vtgate, vgtid, tt)

	conn, err := vtgateconn.Dial(ctx, *vtgate)
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	var fields []*querypb.Field
	var rowEvents []*binlogdatapb.RowEvent
outer:
	for {
		reader, err := conn.VStream(ctx, tt, vgtid, nil)
		if err != nil {
			log.Fatal("error at %v", vgtid)
		}
		for {
			events, err := reader.Recv()
			if err != nil {
				fmt.Printf("remote error: %v at %v, retrying in 1s\n", err, vgtid)
				time.Sleep(1 * time.Second)
				continue outer
			}
			for _, e := range events {
				switch e.Type {
				case binlogdatapb.VEventType_VGTID:
					vgtid = e.Vgtid
				case binlogdatapb.VEventType_FIELD:
					fields = e.FieldEvent.Fields
				case binlogdatapb.VEventType_ROW:
					rowEvents = append(rowEvents, e.RowEvent)
				case binlogdatapb.VEventType_COMMIT:
					fmt.Printf("\nEvent log timestamp: %v --> %v\n", e.Timestamp, time.Unix(e.Timestamp, 0).UTC())
					printRowEvents(vgtid, rowEvents, fields)
					rowEvents = nil
				}
			}
		}
	}
}

func find(re *binlogdatapb.RowEvent, fields []*querypb.Field, id int64) bool {
	for _, change := range re.RowChanges {
		if match(change.Before, fields, id) {
			return true
		}
		if match(change.After, fields, id) {
			return true
		}
	}
	return false
}

func match(p3r *querypb.Row, fields []*querypb.Field, id int64) bool {
	if p3r == nil {
		return false
	}
	p3qr := &querypb.QueryResult{
		Fields: fields,
		Rows:   []*querypb.Row{p3r},
	}
	qr := sqltypes.Proto3ToResult(p3qr)
	if len(qr.Rows) == 0 || len(qr.Rows[0]) == 0 {
		return false
	}
	got, err := sqltypes.ToInt64(qr.Rows[0][0])
	if err != nil {
		return false
	}
	return got == id
}

func printRowEvents(vgtid *binlogdatapb.VGtid, rowEvents []*binlogdatapb.RowEvent, fields []*querypb.Field) {
	for _, re := range rowEvents {
		result := &sqltypes.Result{
			Fields: append([]*querypb.Field{{Name: "table", Type: querypb.Type_VARBINARY}, {Name: "op", Type: querypb.Type_VARBINARY}}, fields...),
		}
		for _, change := range re.RowChanges {
			typ := "U"
			if change.Before == nil {
				typ = "I"
			} else if change.After == nil {
				typ = "D"
			}
			switch typ {
			case "U":
				p3qr := &querypb.QueryResult{
					Fields: fields,
					Rows:   []*querypb.Row{change.Before, change.After},
				}
				qr := sqltypes.Proto3ToResult(p3qr)
				newRow := append([]sqltypes.Value{sqltypes.NewVarBinary(re.TableName), sqltypes.NewVarBinary("BEFORE")}, qr.Rows[0]...)
				result.Rows = append(result.Rows, newRow)
				newRow = append([]sqltypes.Value{sqltypes.NewVarBinary(re.TableName), sqltypes.NewVarBinary("AFTER")}, qr.Rows[1]...)
				result.Rows = append(result.Rows, newRow)
			case "I":
				p3qr := &querypb.QueryResult{
					Fields: fields,
					Rows:   []*querypb.Row{change.After},
				}
				qr := sqltypes.Proto3ToResult(p3qr)
				newRow := append([]sqltypes.Value{sqltypes.NewVarBinary(re.TableName), sqltypes.NewVarBinary("INSERT")}, qr.Rows[0]...)
				result.Rows = append(result.Rows, newRow)
			case "D":
				p3qr := &querypb.QueryResult{
					Fields: fields,
					Rows:   []*querypb.Row{change.Before},
				}
				qr := sqltypes.Proto3ToResult(p3qr)
				newRow := append([]sqltypes.Value{sqltypes.NewVarBinary(re.TableName), sqltypes.NewVarBinary("DELETE")}, qr.Rows[0]...)
				result.Rows = append(result.Rows, newRow)
			}
		}
		printQueryResult(os.Stdout, result)
	}
	fmt.Printf("VGTID after event:  %v\n", vgtid)
}

// printQueryResult will pretty-print a QueryResult to the logger.
// Copied from vtctl/query.go
func printQueryResult(writer io.Writer, qr *sqltypes.Result) {
	table := tablewriter.NewWriter(writer)
	table.SetAutoFormatHeaders(false)

	// Make header.
	header := make([]string, 0, len(qr.Fields))
	for _, field := range qr.Fields {
		header = append(header, field.Name)
	}
	table.SetHeader(header)

	// Add rows.
	for _, row := range qr.Rows {
		vals := make([]string, 0, len(row))
		for _, val := range row {
			vals = append(vals, val.ToString())
		}
		table.Append(vals)
	}

	// Print table.
	table.Render()
}
