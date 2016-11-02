package mdb

import (
	"database/sql"
	"github.com/tjcelaya/mgr8/mutil"
	"strings"
	"errors"
	"log"
	"fmt"
)

type AlterExecutionPlan struct {
	dbName, tableName, colName, colType, newColType string
	autoInc, nullable                               mutil.BinaryChangeIntent
}

func NewAlterExecutionPlan(
dbName,
tableName,
colName,
colType,
newColType string,
autoInc, nullable mutil.BinaryChangeIntent) AlterExecutionPlan {

	return AlterExecutionPlan{dbName, tableName, colName, colType, newColType, autoInc, nullable}
}

func (aep *AlterExecutionPlan) Build(db *sql.DB) ([]AlterStatement, error) {

	if aep.tableName == "" && aep.colName == "" {
		return nil, errors.New("no table or column specified!")
	}

	var whereClauses []string

	whereClauses = append(whereClauses, " TABLE_SCHEMA = '"+aep.dbName+"'")

	if aep.tableName != "" {
		whereClauses = append(whereClauses, " AND TABLE_NAME = '"+aep.tableName+"'")
	}

	if strings.ContainsRune(aep.colName, ',') {
		colNames := make([]string, 0)

		for _, c := range strings.Split(aep.colName, ",") {
			colNames = append(colNames, strings.Trim(c, `'"`))
		}

		whereClauses = append(whereClauses, " AND COLUMN_NAME IN ('"+strings.Join(colNames, "','")+"')")
	} else if aep.colName != "" {
		whereClauses = append(whereClauses, " AND COLUMN_NAME LIKE '"+aep.colName+"'")
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

	if aep.colType != "" {
		whereClauses = append(whereClauses, " AND COLUMN_TYPE LIKE '"+aep.colType+"%'")
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
        WHERE ` + strings.Join(whereClauses, "\n        ")

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
		fmtStr := "CHANGE `%s` `%s` %s "
		alterColType := c.colType

		if aep.newColType != "" {
			alterColType = aep.newColType
		}

		if c.unsigned {
			alterColType = alterColType + " UNSIGNED"
		}

		switch {
		case aep.nullable == mutil.IntentRemove:
			fmtStr = fmtStr + " NOT NULL "
		case aep.nullable == mutil.IntentAdd:
			fmtStr = fmtStr + " NULL "
		case aep.nullable == mutil.IntentNone && !c.nullable == false:
			fmtStr = fmtStr + " NOT NULL "
		case aep.nullable == mutil.IntentNone && c.nullable:
			fmtStr = fmtStr + " NULL "
		}

		switch {
		case aep.autoInc == mutil.IntentAdd:
			fmtStr = fmtStr + " AUTO_INCREMENT "
		case aep.autoInc == mutil.IntentNone && c.autoInc:
			fmtStr = fmtStr + " AUTO_INCREMENT "
		}

		changeStr := fmt.Sprintf(
			fmtStr,
			c.colName,
			c.colName,
			alterColType)

		alters = append(alters, NewAlterStatement(c.tableName, changeStr))
	}

	return alters, nil
}

func CombineSameTableAlters(originalAlters []AlterStatement) []AlterStatement {

	altersGroupedByTable := make(map[string][]AlterStatement, 0)

	for _, a := range originalAlters {
		matchingTableAlters := altersGroupedByTable[a.tableName]
		altersGroupedByTable[a.tableName] = append(matchingTableAlters, a)
	}

	consolidatedAlters := make([]AlterStatement, 0, len(altersGroupedByTable))

	for tableName, combinableAlters := range altersGroupedByTable {
		changeStrs := make([]string, 0, len(combinableAlters))

		for _, a := range combinableAlters {
			changeStrs = append(changeStrs, a.changeStr)
		}

		newAlter := AlterStatement{tableName, strings.Join(changeStrs, ", ")}

		consolidatedAlters = append(consolidatedAlters, newAlter)
	}

	log.Println(len(originalAlters), len(consolidatedAlters))

	return consolidatedAlters
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

		colDef.autoInc = strings.Contains(extraStr, "auto_increment")

		colDef.unsigned = strings.Contains(colDef.colType, "unsigned")

		colDef.nullable = nullableStr == "YES"

		if err != nil {
			return colDefs, err
		}

		colDefs = append(colDefs, colDef)
	}

	return colDefs, nil
}

