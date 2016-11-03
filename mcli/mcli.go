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
	"github.com/tjcelaya/mgr8/mutil"
)

func Run(ior *bufio.Reader, iow io.Writer) int {
	mf := parseFlags()
	shouldExitEarly, exitCode := configureCliAndMaybeExit(iow, mf)

	if shouldExitEarly {
		return exitCode
	}

	db, err := buildDbConnection(mf)

	if err != nil {
		return 1
	}

	var autoIncIntent mutil.BinaryChangeIntent
	switch {
	case *mf.forAutoIncAdd:
		autoIncIntent = mutil.IntentAdd
	case *mf.forAutoIncRemoval:
		autoIncIntent = mutil.IntentRemove
	}

	aep := mdb.NewAlterExecutionPlan(
		*mf.dbName,
		*mf.tableName,
		*mf.colName,
		*mf.typeName,
		*mf.newTypeName,
		autoIncIntent,
		mutil.IntentNone)

	stmts, err := aep.Build(db)

	if err != nil {
		log.Fatalf("failed to build execution plan! %v\n", err)
		return 2
	}

	stmts = mdb.CombineSameTableAlters(stmts)

	if len(stmts) == 0 {
		log.Println("nothing to do?")
		return 0
	} else if *mf.maxDbConn == 0 || len(stmts) < (*mf.maxDbConn) {
		// zero max connections means "one-connection-per-task"
		// also, we shouldn't have more workers than jobs
		*mf.maxDbConn = len(stmts)
		db.SetMaxOpenConns(*mf.maxDbConn)
	}

	if (false == *mf.write) {
		log.Println("exiting early since write mode is disabled")
		for _, a := range stmts {
			log.Println(" -- " +  a.Serialize())
		}
		return 0
	}

	stmtCh := make(chan mdb.AlterStatement, *mf.maxDbConn)
	doneCh := make(chan mdb.AlterResult, *mf.maxDbConn)
	defer close(stmtCh)
	defer close(doneCh)

	log.Printf("kicking off %d workers for %d statements\n", *mf.maxDbConn, len(stmts))

	for i := 0; i < *mf.maxDbConn; i++ {
		go queryWorker(db, stmtCh, doneCh)
	}

	for i := 0; i < len(stmts); i++ {
		go func(clsI int) {
			stmtCh <- stmts[clsI]
		}(i)
	}

	for i := 0; i < len(stmts); i++ {
		// log.Printf("waiting for %#v\n", i)
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

var (
	Buildstamp string
)

func configureCliAndMaybeExit(iow io.Writer, mf MFlags) (bool, int) {
	log.SetOutput(iow)
	log.SetPrefix("mgr8 - ")
	log.Println("Build: " + Buildstamp)

	if len(os.Args) == 1 {
		log.Println("nothing to do? try -help !")
		return true, 0
	}

	if *mf.dbName == "" {
		log.Println("no db given!")
		return true, 1
	}

	log.Printf("database: %s", *mf.dbName)

	if *mf.tableName == "" && *mf.colName == "" {
		log.Println("no table or column given, need at least one!")
		return true, 1
	} else if *mf.colName == "" {
		log.Println("no table operations supported yet")
		return true, -1
	} else if *mf.tableName == "" {
		log.Printf("column: %s", *mf.colName)
	}

	if *mf.forAutoIncAdd && *mf.forAutoIncRemoval {
		log.Println("can't both add and remove auto-inc!")
		return true, 2
	} else if *mf.forAutoIncAdd {
		log.Println("auto_inc: add")
	} else if *mf.forAutoIncRemoval {
		log.Println("auto_inc: remove")
	}

	if *mf.newTypeName != "" {
		log.Printf("column type change: %s\n", *mf.newTypeName)
	}

	return false, 0
}

func buildDbConnection(mf MFlags) (*sql.DB, error) {
	var passwordProvided string

	if *mf.askForPass {
		passwordProvided = prompt.Password("password pls")
	} else {
		passwordProvided = *mf.pass
	}

	db, err := mdb.New(fmt.Sprintf("%s:%s@/%s", *mf.user, passwordProvided, *mf.dbName), *mf.maxDbConn)

	if err != nil {
		log.Fatalf("couldnt connect to the database! %v\n", err)
		return nil, err
	}

	log.Println("connected!")

	return db, nil
}

func queryWorker(db *sql.DB, stmtCh chan mdb.AlterStatement, resCh chan mdb.AlterResult) {
	for {
		stmt := <-stmtCh
		resCh <- stmt.Apply(db)
	}
}
