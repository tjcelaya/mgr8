package main

import "fmt"
import "database/sql"
import (
	"flag"
	_ "github.com/go-sql-driver/mysql"
	"log"
	"os"
)

func p(xs ...interface{}) {
	for _, x := range xs {
		fmt.Printf("%#v\n", x)
	}
}

func dieOnError(err error) {
	if err != nil {
		log.Fatal(err)
		panic(err.Error())
	}
}

func main() {
	dbname := flag.String("db", "db", "Database name to work on.")
	colname := flag.String("column", "id", "Column name usually 'id'.")

	action := flag.String("action", "", "Whether to add auto_inc to the column")

	var maybe_auto_inc string
	switch *action {
	case "--add":
		maybe_auto_inc = ""
	case "--remove":
		maybe_auto_inc = "AND EXTRA = 'auto_increment' "
	case "":
	default:
		fmt.Println("What am I supposed to do with this?")
		os.Exit(1)
		return
	}

	db, err := sql.Open("mysql", "root:secret@/"+(*dbname))
	dieOnError(err)

	err = db.Ping()
	dieOnError(err)
	defer db.Close()

	rows, err := db.Query(`
		SELECT
			TABLE_NAME, COLUMN_NAME, COLUMN_TYPE
		FROM
			information_schema.COLUMNS
		WHERE
			TABLE_SCHEMA = ?
			AND COLUMN_NAME = ?
		`+maybe_auto_inc,
		dbname,
		colname)

	dieOnError(err)

	row := struct{ TABLE_NAME, COLUMN_NAME, COLUMN_TYPE string }{}
	var numrows int

	for rows.Next() {
		dieOnError(rows.Scan(&row.TABLE_NAME, &row.COLUMN_NAME, &row.COLUMN_TYPE))
		numrows++
		fmt.Printf("%s %s %s\n", row.TABLE_NAME, row.COLUMN_NAME, row.COLUMN_TYPE)
	}

	fmt.Printf("%d results\n", numrows)
	os.Exit(0)
}
