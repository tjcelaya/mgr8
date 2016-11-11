package main

// -column mgr8_test_id.int_id,mgr8_test_another_table_id.nullable_int_id,mgr8_test_id.auto_inc_nonnullable_int_id, -newcoltype bigint -table mgr8_test_id
// -character-set utf8mb4 -collation utf8mb4_general_ci

import (
	"bufio"
	_ "database/sql/driver"
	_ "fmt"
	_ "github.com/go-sql-driver/mysql"
	"github.com/tjcelaya/mgr8/mcli"
	"os"
)

var (
	buildstamp string
)

func main() {
	var run int
	mcli.Buildstamp = buildstamp
	bufioReader := bufio.NewReader(os.Stdin)
	run = mcli.Run(bufioReader, os.Stdout)
	os.Exit(run)
}
