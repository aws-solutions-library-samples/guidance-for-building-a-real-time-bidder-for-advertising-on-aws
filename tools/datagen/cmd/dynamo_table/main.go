package main

import (
	"datagen/pkg/generator"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/kelseyhightower/envconfig"
)

var (
	table     = flag.String("table", "", "table name")
	pk        = flag.String("pk", "", "partition key (used if creating a table)")
	pkType    = flag.String("pk-type", string(types.ScalarAttributeTypeB), "partition key type (used if creating a table)")
	cmd       = flag.String("cmd", "", "command to execute: one of [create delete describe list]")
	listLimit = flag.Int64("limit", 0, "number of items to list (used when listing a table)")
)

func handleError(err error) {
	fmt.Printf("datagen: %v\n\n", err)
	flag.PrintDefaults()
	os.Exit(1)
}

func main() {
	var err error
	var cfg generator.AWSConfig

	if err = envconfig.Process("", &cfg); err != nil {
		handleError(err)
	}

	flag.Parse()

	if table == nil || strings.Trim(*table, " ") == "" {
		handleError(fmt.Errorf("missing table name"))
	}

	conn := generator.NewTableConn(*table, &cfg)

	var out interface{}

	switch *cmd {
	case "create":
		out, err = conn.CreateTable(*pk, types.ScalarAttributeType(*pkType))
	case "delete":
		out, err = conn.DeleteTable()
	case "describe":
		out, err = conn.DescribeTable()
	case "list":
		out, err = conn.ListTable(int32(*listLimit))
	default:
		handleError(fmt.Errorf("unknown command: %s", *cmd))
	}

	if err != nil {
		handleError(err)
	}

	js, err := json.Marshal(out)
	if err != nil {
		handleError(err)
	}
	fmt.Println(string(js))
}
