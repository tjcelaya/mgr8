package mdb

import (
	"fmt"
	"database/sql"
)

/*

	ALTER STATEMENT

 */

type AlterStatement struct {
	colDef      PrimaryColumnDefinition
	queryString string
}

func NewAlterStatement(cd PrimaryColumnDefinition, qs string) AlterStatement {
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

type AlterExecutionPlan struct {
	dbConn           *sql.DB
	dbName, colName  string
	forAddingAutoInc bool
}

func NewAlterExecutionPlan(dbConn *sql.DB, dbName, colName string, forAddingAutoInc bool) AlterExecutionPlan {
	return AlterExecutionPlan{dbConn, dbName, colName, forAddingAutoInc}
}

func (aep *AlterExecutionPlan) Build(db *sql.DB) ([]AlterStatement, error) {
	var autoIncClause string
	if aep.forAddingAutoInc {
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
		aep.dbName,
		aep.colName)

	if err != nil {
		return nil, err
	}

	colDefs, err := aep.fetchPrimaryColumnDefinitions(rows)
	alters := []AlterStatement{}

	for _, c := range colDefs {
		fmtStr := "ALTER TABLE %s CHANGE `%s` `%s` %s "

		// id columns aren't usually null
		//switch row.nullable {
		//case "YES":
		//	row.nullable = " NULL "
		//case "NO":
		//	row.nullable = " NOT NULL "
		//}

		if aep.forAddingAutoInc {
			fmtStr = fmtStr + " AUTO_INCREMENT "
		}

		str := fmt.Sprintf(
			fmtStr + "\n",
			c.tableName,
			c.colName,
			c.colName,
			c.colType)

		a := NewAlterStatement(c, str)

		alters = append(alters, a)

	}

	return alters, nil
}

func (aep *AlterExecutionPlan) fetchPrimaryColumnDefinitions(rows *sql.Rows) (colDefs []PrimaryColumnDefinition, err error) {
	for rows.Next() {
		var colDef PrimaryColumnDefinition
		err = rows.Scan(&colDef.tableName, &colDef.colName, &colDef.colType, &colDef.nullable)

		if err != nil {
			return colDefs, err
		}

		colDefs = append(colDefs, colDef)
	}

	return colDefs, nil
}
