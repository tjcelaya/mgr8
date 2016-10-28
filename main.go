package main

import (
	"bufio"
	_ "database/sql/driver"
	_ "fmt"
	_ "github.com/go-sql-driver/mysql"
	"github.com/tjcelaya/mgr8/mcli"
	"os"
)

func main() {
	var run int
	bufioReader := bufio.NewReader(os.Stdin)
	run = mcli.Run(bufioReader, os.Stdout)
	os.Exit(run)
}
