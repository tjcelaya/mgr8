package mdb

import (
	"database/sql"
	"fmt"
	"github.com/tjcelaya/mgr8/mutil"
	"log"
	_ "os"
	"strings"
	"github.com/siddontang/go-mysql/Godeps/_workspace/src/github.com/juju/errors"
)

/*

	ALTER STATEMENT

*/

type AlterStatement struct {
	colDef      ColumnDefinition
	queryString string
}

func NewAlterStatement(cd ColumnDefinition, qs string) AlterStatement {
	return AlterStatement{colDef: cd, queryString: qs}
}

func (a *AlterStatement) SetQueryString(s string) {
	a.queryString = s
}

func (a *AlterStatement) Apply(forReal bool, db *sql.DB) (result AlterResult) {
	if !forReal {
		return AlterResult{alter: *a, rowsAffected: -1, err: nil}
	}

	fmt.Println("working on " + a.colDef.tableName)
	res, err := db.Exec(a.queryString)
	fmt.Println("finished " + a.colDef.tableName)

	if err != nil {
		result = AlterResult{alter: *a, rowsAffected: -1, err: err}
		fmt.Printf("error occured with %s: %#v\n",
			a.queryString,
			result.err)
		return result
	}

	rowsAffected, err := res.RowsAffected()

	if err != nil {
		result = AlterResult{alter: *a, rowsAffected: -1, err: err}
		fmt.Printf("error occured getting affected rows with %s: %#v\n",
			a.queryString,
			result.err)
		return result
	}

	return AlterResult{alter: *a, rowsAffected: int(rowsAffected), err: nil}
}

/*

	ALTER RESULT

*/

type AlterResult struct {
	alter        AlterStatement
	rowsAffected int
	err          error
}

func (a *AlterResult) Err() error {
	return a.err
}

func (a *AlterResult) TargetIdentifier() string {
	return a.alter.colDef.tableName
}

func (a *AlterResult) ResultCount() int {
	return a.rowsAffected
}

func (a *AlterResult) PlanDescription() string {
	return a.alter.queryString
}

/*

	ALTER EXECUTION PLAN

*/

type AlterError string

type AlterExecutionPlan struct {
	dbName, tableName, colName, typeName string
	autoInc, nullable                    mutil.BinaryChangeIntent
}

func NewAlterExecutionPlan(
	dbName,
	tableName,
	colName,
	typeName string,
	autoInc, nullable mutil.BinaryChangeIntent) AlterExecutionPlan {

	return AlterExecutionPlan{dbName, tableName, colName, typeName, autoInc, nullable}
}

func (aep *AlterExecutionPlan) Build(db *sql.DB) ([]AlterStatement, error) {

	if aep.tableName == "" && aep.colName == "" {
		return nil, errors.New("no table or column specified!")
	}

	var whereClauses []string

	whereClauses = append(whereClauses, " TABLE_SCHEMA = '" + aep.dbName+"'")

	if aep.tableName != "" {
		whereClauses = append(whereClauses, " AND TABLE_NAME = " + aep.tableName)
	}

	if strings.ContainsRune(aep.colName, ',') {
		colNames := make([]string, 0)

		for _, c := range strings.Split(aep.colName, ",") {
			colNames = append(colNames, strings.Trim(c, `'"`))
		}

		whereClauses = append(whereClauses, " AND COLUMN_NAME IN ('" + strings.Join(colNames, "','") + "')")
	} else if aep.colName != "" {
		whereClauses = append(whereClauses, " AND COLUMN_NAME LIKE '" + aep.colName + "'")
	}

	switch aep.autoInc {
	case mutil.IntentAdd:
		whereClauses = append(whereClauses, " AND EXTRA = '' ")
	case mutil.IntentRemove:
		whereClauses = append(whereClauses, " AND EXTRA = 'auto_increment' ")
	}

	switch aep.nullable {
	case mutil.IntentAdd:
		whereClauses = append(whereClauses, " AND IS_NULLABLE = 'NO' ")
	case mutil.IntentRemove:
		whereClauses = append(whereClauses, " AND IS_NULLABLE = 'YES' ")
	}

	if aep.typeName != "" {
		whereClauses = append(whereClauses, " AND COLUMN_TYPE NOT LIKE '" + aep.typeName + "%'")
	}

	colQuery := `
		SELECT
			TABLE_SCHEMA dbName,
			TABLE_NAME tableName,
			COLUMN_NAME colName,
			COLUMN_TYPE colType,
			IS_NULLABLE nullable,
			EXTRA extra
		FROM
			information_schema.COLUMNS
		WHERE ` + strings.Join(whereClauses, "\n\t\t")

	log.Println("running " + colQuery)

	rows, err := db.Query(colQuery)

	if err != nil {
		log.Printf("error building plan: %v", err)
		return nil, err
	}

	colDefs, err := aep.hydrateColumnDefinitions(rows)

	if err != nil {
		log.Printf("error hydrating rows: %v", err)
		return nil, err
	}

	if len(colDefs) == 0 {
		return nil, nil
	}

	alters := []AlterStatement{}

	for _, c := range colDefs {
		fmtStr := "ALTER TABLE %s CHANGE `%s` `%s` %s "
		alterColType := c.colType

		if aep.typeName != "" {
			alterColType = aep.typeName
		}

		if aep.autoInc == mutil.IntentAdd {
			fmtStr = fmtStr + " AUTO_INCREMENT "
		}

		str := fmt.Sprintf(
			fmtStr+"\n",
			c.tableName,
			c.colName,
			c.colName,
			alterColType)

		alters = append(alters, NewAlterStatement(c, str))
	}

	return alters, nil
}

func (aep *AlterExecutionPlan) hydrateColumnDefinitions(rows *sql.Rows) (colDefs []ColumnDefinition, err error) {
	for rows.Next() {
		var colDef ColumnDefinition
		var extraStr, nullableStr string
		err = rows.Scan(
			&colDef.dbName,
			&colDef.tableName,
			&colDef.colName,
			&colDef.colType,
			&nullableStr,
			&extraStr)

		if err != nil {
			return colDefs, err
		}

		colDefs = append(colDefs, colDef)
	}

	return colDefs, nil
}
