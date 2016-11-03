package main

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
