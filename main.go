package main

import "fmt"
import "database/sql"
import (
	_ "database/sql/driver"
	"flag"
	_ "github.com/go-sql-driver/mysql"
	"os"
	"github.com/youtube/vitess/go/sync2"
)

var (
	performExec = true
)

func p(xs ...interface{}) {
	for _, x := range xs {
		fmt.Printf("%#v\n", x)
	}
}

func dieOnError(err error) {
	if err != nil {
		panic(err.Error())
	}
}

func getDatabaseConnection(user, pass, dbName string, maxConn int) *sql.DB {
	db, err := sql.Open("mysql", fmt.Sprintf("%s:%s@/%s", user, pass, dbName))
	dieOnError(err)
	err = db.Ping()
	dieOnError(err)
	db.SetMaxOpenConns(maxConn)
	db.SetMaxIdleConns(maxConn)

	return db
}

func main() {
	dbname := flag.String("db", "db", "Database name to work on.")
	colname := flag.String("column", "id", "Column name usually 'id'.")
	user := flag.String("user", "root", "username")
	pass := flag.String("pass", "secret", "password")
	maxConn := flag.Int("connections", 5, "Number of connections to use")
	forAdd := flag.Bool("add", false, "whether to add auto_increment")
	forRemoval := flag.Bool("remove", false, "whether to remove auto_increment")
	flag.Parse()
	//actiona :=
	 getAction(forAdd, forRemoval)

	//fmt.Println(action + (*colname) + " column in " + (*dbname))

	db := getDatabaseConnection(*user, *pass, *dbname, *maxConn)
	defer db.Close()

	stmts := buildAlterStatements(db, *dbname, *colname, *forAdd)
	if len(stmts) < (*maxConn) {
		*maxConn = len(stmts)
	}

	if len(stmts) == 0 {
		fmt.Println("nothing to do")
		os.Exit(1)
	}

	//wg := sync.WaitGroup{}
	stmtCh := make(chan AlterStatement, *maxConn)
	doneCh := make(chan AlterResult, *maxConn)
	concurrentAlters := sync2.NewAtomicInt32(0)

	fmt.Printf("kicking off %d workers for %d statements\n", *maxConn, len(stmts))
	for i := 0; i < *maxConn - 1; i++ {
		go queryWorker(db, stmtCh, doneCh, concurrentAlters)
	}

	for i := 0; i < len(stmts); i++ {
		go func(clsI int) {
			stmtCh <- stmts[clsI]
		}(i)
	}

	for i := 0; i < len(stmts); i++ {
		completionResult := <-doneCh
		concurrentAlters.Add(1)
		//fmt.Println("<<< m2 " + fmt.Sprintf("%#v", i) + " -> " + stmts[i].colDef.tableName)

		if completionResult.err != nil {
			fmt.Printf("error running query on table %s complete: %s \n",
				completionResult.alter.colDef.tableName,
				completionResult.alter.queryString)
			//fmt.Println(completionResult.err)
			continue
		}

		fmt.Printf("table %s complete, %d rows\n",
			completionResult.alter.colDef.tableName,
			completionResult.rowsAffected)
	}

	//fmt.Println("we should be done? " + fmt.Sprintf("%#v", stmtsCompleted.Get()))
	os.Exit(0)
}
func getAction(add *bool, remove *bool) string {
	var action string
	if *add == false && *remove == false {
		//fmt.Println("What am I supposed to do with this?")
		os.Exit(1)
	} else if *remove {
		action = "Removing auto_inc from "
	} else if *add {
		action = "Adding auto_inc to "
	}

	return action
}

type PrimaryColumnDefinition struct {
	tableName, colName, colType, nullable string
}

type AlterStatement struct {
	colDef      PrimaryColumnDefinition
	queryString string
}

type AlterResult struct {
	alter        AlterStatement
	rowsAffected int
	err          error
}

func buildAlterStatements(
db *sql.DB,
dbName, colName string,
forAddingAuto bool) []AlterStatement {

	var autoIncClause string
	if forAddingAuto {
		autoIncClause = " AND EXTRA = '' "
	} else {
		autoIncClause = " AND EXTRA = 'auto_increment' "
	}

	strstr := `
		SELECT
			TABLE_NAME tableName,
			COLUMN_NAME colName,
			COLUMN_TYPE colType,
			IS_NULLABLE nullable
		FROM
			information_schema.COLUMNS
		WHERE
			TABLE_SCHEMA = ?
			AND COLUMN_NAME = ?
		` + autoIncClause

	rows, err := db.Query(strstr,
		dbName,
		colName)

	dieOnError(err)

	alters := []AlterStatement{}
	var numrows int

	for rows.Next() {
		var alter AlterStatement
		var row PrimaryColumnDefinition
		dieOnError(rows.Scan(&row.tableName, &row.colName, &row.colType, &row.nullable))
		numrows++

		alter.colDef = row

		fmtStr := "ALTER TABLE %s CHANGE `%s` `%s` %s "

		// id columns aren't usually null
		//switch row.nullable {
		//case "YES":
		//	row.nullable = " NULL "
		//case "NO":
		//	row.nullable = " NOT NULL "
		//}

		if forAddingAuto {
			fmtStr = fmtStr + " AUTO_INCREMENT "
		}

		alter.queryString = fmt.Sprintf(
			fmtStr + "\n",
			alter.colDef.tableName,
			alter.colDef.colName,
			alter.colDef.colName,
			alter.colDef.colType)

		alters = append(alters, alter)
	}

	return alters
}

func queryWorker(db *sql.DB, stmtCh chan AlterStatement, resCh chan AlterResult, anInt sync2.AtomicInt32) {
	for {
		//fmt.Println(" worker blocked ")
		stmt := <-stmtCh
		//fmt.Println(" WOOt IT's TIME FOR " + stmt.colDef.tableName)
		anInt.Add(1)
		alter := applyAlter(db, stmt)
		//fmt.Println(" + worker done! trying to submit result ")
		resCh <- alter
		//fmt.Println(" - result submitted! " + stmt.colDef.tableName)
	}
}

func applyAlter(db *sql.DB, alter AlterStatement) AlterResult {

	if !performExec {
		return AlterResult{alter, -1, nil}
	}

	fmt.Println("working on " + alter.colDef.tableName)
	res, err := db.Exec(alter.queryString)
	fmt.Println("finished " + alter.colDef.tableName)

	result := AlterResult{}

	if err != nil {
		result = AlterResult{alter, -1, err}
		fmt.Printf("error occured with %s: %#v\n",
			result.alter.queryString,
			result.err)
		return result
	}

	rowsAffected, err := res.RowsAffected()

	if err != nil {
		result = AlterResult{alter, -1, err}
		fmt.Printf("error occured getting affected rows with %s: %#v\n",
			result.alter.queryString,
			result.err)
		return result
	}

	result = AlterResult{alter, int(rowsAffected), nil}
	//fmt.Println(result)
	return result
}
