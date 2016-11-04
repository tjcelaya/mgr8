package mdb

import (
	"database/sql"
	"github.com/tjcelaya/mgr8/mutil"
	"strings"
	"errors"
	"log"
	_"fmt"
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
		whereClauses = append(whereClauses, "TABLE_NAME = '"+aep.tableName+"'")
	}

	if strings.ContainsRune(aep.colName, ',') {
		whereColClauses := make([]string, 0)
		compoundColumnNamesUsed := strings.ContainsRune(aep.colName, '.')
		cols := strings.Split(aep.colName, ",")

		for _, c := range cols {

			if 0 == len(c) {
				continue
			}

			if compoundColumnNamesUsed {
				columnNameParts := strings.Split(c, ".")

				if len(columnNameParts) != 2 {
					return nil, errors.New("invalid mostly-qualified column name (all columns must have table names if any have table names) : " + c)
				}

				whereColClauses = append(whereColClauses, fmt.Sprintf("(TABLE_NAME = '%s' AND COLUMN_NAME ='%s' )", columnNameParts[0], columnNameParts[1]))

			} else {
				whereColClauses = append(whereColClauses, "COLUMN_NAME = '" + strings.Trim(c, `'"`), "' ")
			}
		}

		whereClauses = append(whereClauses, strings.Join(whereColClauses, " OR "))
	} else if aep.colName != "" {
		whereClauses = append(whereClauses, "COLUMN_NAME LIKE '"+aep.colName+"'")
	}

	switch aep.autoInc {
	case mutil.IntentAdd:
		whereClauses = append(whereClauses, "EXTRA = '' ")
	case mutil.IntentRemove:
		whereClauses = append(whereClauses, "EXTRA = 'auto_increment' ")
	}

	switch aep.nullable {
	case mutil.IntentAdd:
		whereClauses = append(whereClauses, "IS_NULLABLE = 'NO' ")
	case mutil.IntentRemove:
		whereClauses = append(whereClauses, "IS_NULLABLE = 'YES' ")
	}

	if aep.colType != "" {
		whereClauses = append(whereClauses, "COLUMN_TYPE LIKE '"+aep.colType+"%'")
	}

	colQuery := buildColumnDefinitionQuery() + strings.Join(whereClauses, "\n        AND ")

	log.Println("running " + colQuery)

	rows, err := db.Query(colQuery)

	if err != nil {
		log.Printf("error building plan: %v", err)
		return nil, err
	}

	colDefs, err := hydrateColumnDefinitions(rows)

	if err != nil {
		log.Printf("error hydrating rows: %v", err)
		return nil, err
	}

	if len(colDefs) == 0 {
		return nil, nil
	}

	alters := []AlterStatement{}

	for _, c := range colDefs {
		alterColClauses := []string{fmt.Sprintf("CHANGE `%s` `%s`", c.colName, c.colName)}

		// type
		alterColType := c.colType
		newTypeIsStringType := isStringType(alterColType)

		if aep.newColType != "" {
			alterColType = aep.newColType
		}

		if c.isUnsigned {
			alterColType = alterColType + " UNSIGNED"
		}

		alterColClauses = append(alterColClauses, alterColType)

		if newTypeIsStringType {
			// keep charset and collation for string types
			alterColClauses = append(alterColClauses, c.characterSetName)
			alterColClauses = append(alterColClauses, c.collationName)
		}

		// [not] null
		if aep.nullable == mutil.IntentRemove ||
			(aep.nullable == mutil.IntentNone && c.isNullable == false) {
			alterColClauses = append(alterColClauses, "NOT NULL")
		} else if aep.nullable == mutil.IntentNone && c.isNullable == true {
			alterColClauses = append(alterColClauses, "NULL")
		}

		// add default if present
		if c.hasDefault {
			alterColClauses = append(alterColClauses, "DEFAULT " +c.defaultValue)
		}

		// extra
		if aep.autoInc == mutil.IntentAdd || aep.autoInc == mutil.IntentNone && c.autoInc {
			alterColClauses = append(alterColClauses, "AUTO_INCREMENT")
		}

		changeStr := strings.Join(alterColClauses, " ")

		alter := NewAlterStatement(c.tableName, changeStr)

		alters = append(alters, alter)
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

		newAlter := AlterStatement{tableName, strings.Join(changeStrs, ",\n    ") + "\n;"}

		consolidatedAlters = append(consolidatedAlters, newAlter)
	}

	log.Printf("consolidated %d individual column alters into %d table alters", len(originalAlters), len(consolidatedAlters))

	return consolidatedAlters
}
