package main

import (
	_ "database/sql/driver"
	_ "github.com/go-sql-driver/mysql"
	"migorate/mcli"
	"os"
)

func main() {
	os.Exit(mcli.Run(os.Stdout, os.Stdin))
}
