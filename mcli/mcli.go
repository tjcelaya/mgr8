package mcli

import "fmt"
import "database/sql"
import (
	_ "database/sql/driver"
	_ "github.com/go-sql-driver/mysql"
	"io"
	"migorate/mdb"
	"os"
)

var (
	forReal = true
)

func Run(ior io.Reader, iow io.Writer) int {
	mf := parseFlags()
	action := getAction(mf.forAdd, mf.forRemoval)
	fmt.Printf("%s column %s from database %s", action, *mf.colName, *mf.dbName)
	db, err := mdb.New(fmt.Sprintf("%s:%s@/%s", *mf.user, *mf.pass, *mf.dbName), *mf.maxDbConn)

	if err != nil {
		return 1
	}

	defer db.Close()
	aep := mdb.NewAlterExecutionPlan(db, *mf.dbName, *mf.colName, *mf.forAdd)
	stmts, err := aep.Build(db)

	if err != nil {
		return 2
	}

	if len(stmts) == 0 {
		fmt.Println("nothing to do")
		return 1
	} else if *mf.maxDbConn == 0 || len(stmts) < (*mf.maxDbConn) {
		// zero max connections means "one-connection-per-table"
		// also, we shouldn't have more workers than jobs
		*mf.maxDbConn = len(stmts)
		db.SetMaxOpenConns(*mf.maxDbConn)
	}

	stmtCh := make(chan mdb.AlterStatement, *mf.maxDbConn)
	doneCh := make(chan mdb.AlterResult, *mf.maxDbConn)
	defer close(stmtCh)
	defer close(doneCh)

	fmt.Printf("kicking off %d workers for %d statements\n", *mf.maxDbConn, len(stmts))
	for i := 0; i < *mf.maxDbConn; i++ {
		go queryWorker(db, stmtCh, doneCh, forReal)
	}
	for i := 0; i < len(stmts); i++ {
		go func(clsI int) {
			stmtCh <- stmts[clsI]
		}(i)
	}
	for i := 0; i < len(stmts); i++ {
		fmt.Printf("waiting for %#v\n", i)
		completionResult := <-doneCh
		if completionResult.Err() != nil {
			fmt.Printf("error running query on table %s complete: %s \n",
				completionResult.TargetIdentifier(),
				completionResult.PlanDescription())
		}
		fmt.Printf("table %s complete, %d rows\n",
			completionResult.TargetIdentifier(),
			completionResult.ResultCount())
	}
	return 0
}
func getAction(add *bool, remove *bool) string {
	var action string
	if *add == false && *remove == false {
		os.Exit(1)
	} else if *remove {
		action = "Removing auto_inc from "
	} else if *add {
		action = "Adding auto_inc to "
	}
	return action
}

func queryWorker(db *sql.DB, stmtCh chan mdb.AlterStatement, resCh chan mdb.AlterResult, forReal bool) {
	for {
		stmt := <-stmtCh
		resCh <- stmt.Apply(forReal, db)
	}
}
