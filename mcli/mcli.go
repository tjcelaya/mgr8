package mcli

import "fmt"
import "database/sql"
import (
	"bufio"
	_ "database/sql/driver"
	_ "github.com/go-sql-driver/mysql"
	"github.com/segmentio/go-prompt"
	"github.com/tjcelaya/mgr8/mdb"
	"io"
	"log"
	"os"
)

var (
	forReal = true
)

func Run(ior *bufio.Reader, iow io.Writer) int {
	mf := parseFlags()

	log.SetOutput(iow)
	log.SetPrefix("mgr8 - ")

	if len(os.Args) == 1 {
		log.Printf("\nmy name is %s and I have nothing to do? try -help !\n", os.Args[0])
		return 0
	}

	var passwordProvided string

	if *mf.askForPass {
		passwordProvided = prompt.Password("password pls")
	} else {
		passwordProvided = *mf.pass
	}

	log.Printf(
		"%s column %s in database %s",
		getAction(mf.forAutoIncAdd, mf.forAutoIncRemoval),
		*mf.colName,
		*mf.dbName)

	db, err := mdb.New(fmt.Sprintf("%s:%s@/%s", *mf.user, passwordProvided, *mf.dbName), *mf.maxDbConn)

	if err != nil {
		log.Fatalf("couldnt connect to the database! %v\n", err)
		return 1
	}

	log.Printf("connected!\n")

	defer db.Close()
	aep := mdb.NewAlterExecutionPlan(db, *mf.dbName, *mf.colName, *mf.forAutoIncAdd)
	stmts, err := aep.Build(db)

	if err != nil {
		log.Fatalf("failed to build execution plan! %v\n", err)
		return 2
	}

	if len(stmts) == 0 {
		log.Println("nothing to do?")
		return 0
	} else if *mf.maxDbConn == 0 || len(stmts) < (*mf.maxDbConn) {
		// zero max connections means "one-connection-per-task"
		// also, we shouldn't have more workers than jobs
		*mf.maxDbConn = len(stmts)
		db.SetMaxOpenConns(*mf.maxDbConn)
	}

	stmtCh := make(chan mdb.AlterStatement, *mf.maxDbConn)
	doneCh := make(chan mdb.AlterResult, *mf.maxDbConn)
	defer close(stmtCh)
	defer close(doneCh)

	log.Printf("kicking off %d workers for %d statements\n", *mf.maxDbConn, len(stmts))

	for i := 0; i < *mf.maxDbConn; i++ {
		go queryWorker(db, stmtCh, doneCh, forReal)
	}
	for i := 0; i < len(stmts); i++ {
		go func(clsI int) {
			stmtCh <- stmts[clsI]
		}(i)
	}
	for i := 0; i < len(stmts); i++ {
		log.Printf("waiting for %#v\n", i)
		completionResult := <-doneCh
		if completionResult.Err() != nil {
			log.Printf("error running query on table %s complete: %s \n",
				completionResult.TargetIdentifier(),
				completionResult.PlanDescription())
		}
		log.Printf("table %s complete, %d rows\n",
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
		action = "Removing auto_inc from"
	} else if *add {
		action = "Adding auto_inc to"
	}
	return action
}

func queryWorker(db *sql.DB, stmtCh chan mdb.AlterStatement, resCh chan mdb.AlterResult, forReal bool) {
	for {
		stmt := <-stmtCh
		resCh <- stmt.Apply(forReal, db)
	}
}
