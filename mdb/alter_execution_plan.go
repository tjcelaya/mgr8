package mdb

import (
	"database/sql"
	"errors"
	"fmt"
	_ "fmt"
	"github.com/tjcelaya/mgr8/mutil"
	"log"
	"strings"
)

type AlterExecutionPlan struct {
	dbName, tableName, colName, colType, newColType, newCharacterSet, newCollation string
	autoInc, nullable                                                              mutil.BinaryChangeIntent
}

func NewAlterExecutionPlan(
	dbName,
	tableName,
	colName,
	colType,
	newColType,
	newCharacterSet,
	newCollation string,
	autoInc, nullable mutil.BinaryChangeIntent) AlterExecutionPlan {

	return AlterExecutionPlan{dbName, tableName, colName, colType, newColType, newCharacterSet, newCollation, autoInc, nullable}
}

func (aep *AlterExecutionPlan) Build(db *sql.DB) ([]AlterStatement, error) {

	if aep.tableName == "" && aep.colName == "" {
		return nil, errors.New("no table or column specified!")
	}

	if aep.newCharacterSet != "" || aep.newCollation != "" {
		if (aep.newCharacterSet != "" && aep.newCollation == "") ||
			(aep.newCharacterSet == "" && aep.newCollation != "") {
			return nil, errors.New("character set and collation must be specified together")
		}
	}

	var whereClauses []string

	whereClauses = append(whereClauses, " TABLE_SCHEMA = '"+aep.dbName+"'")

	tableWhereClause, err := aep.buildTableWhereClause()

	if err != nil {
		return nil, err
	}

	whereClauses = append(whereClauses, tableWhereClause...)

	whereColClauses := make([]string, 0)

	potentiallyQualifiedColSyms := mutil.BuildCommaSeparatedQualifiedSymbolList(aep.colName, '.')

	for _, colSymParts := range potentiallyQualifiedColSyms {
		if 2 < len(colSymParts) {
			return nil, errors.New(fmt.Sprintln("strangely qualified colsym: ", colSymParts))
		} else if 2 == len(colSymParts) {
			whereColClauses = append(whereColClauses,
				fmt.Sprintf("(TABLE_NAME = '%s' AND COLUMN_NAME ='%s' )",
					colSymParts[0],
					colSymParts[1]))
		} else {
			whereColClauses = append(whereColClauses, "COLUMN_NAME = '"+strings.Trim(colSymParts[0], `'"`)+"' ")
		}
	}

	whereClauses = append(whereClauses, "("+strings.Join(whereColClauses, " OR ")+")")

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

	colQuery := fmt.Sprint(
		buildColumnDefinitionQuery(),
		strings.Join(whereClauses, "\n        AND "))

	log.Println("running " + colQuery)

	rows, err := db.Query(colQuery)

	if err != nil {
		log.Printf("error querying for targets: %v", err)
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

	var tableAlters, columnAlters []AlterStatement
	tableSet := make(map[string]interface{})

	for _, c := range colDefs {
		if _, ok := tableSet[c.tableName]; !ok {
			tableSet[c.tableName] = nil
		}

		if aep.colName == "" && aep.colType == "" {
			continue // we're only here to collect table names while grouping is figured out
		}

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
			alterColClauses = append(alterColClauses, "DEFAULT "+c.defaultValue)
		}

		// extra
		if aep.autoInc == mutil.IntentAdd || aep.autoInc == mutil.IntentNone && c.autoInc {
			alterColClauses = append(alterColClauses, "AUTO_INCREMENT")
		}

		changeStr := strings.Join(alterColClauses, " ")

		alter := NewAlterStatement(c.tableName, changeStr)

		columnAlters = append(columnAlters, alter)
	}

	if aep.newCharacterSet != "" && aep.newCollation != "" {
		for tableName := range tableSet {
			tableAlters = append(tableAlters, NewAlterStatement(tableName, "CONVERT TO CHARACTER SET "+aep.newCharacterSet+" COLLATE "+aep.newCollation))
		}
	}

	return combineSameTableAlters(append(tableAlters, columnAlters...)), nil
}

func combineSameTableAlters(originalAlters []AlterStatement) []AlterStatement {

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

	log.Printf("consolidated %d individual alters into %d grouped table alters", len(originalAlters), len(consolidatedAlters))

	return consolidatedAlters
}

func (aep *AlterExecutionPlan) buildTableWhereClause() ([]string, error) {

	if aep.tableName == "" {
		return []string{}, nil
	}

	potentiallyQualifiedTableSyms := mutil.BuildCommaSeparatedQualifiedSymbolList(aep.tableName, '.')
	tableWhereClauses := make([]string, 0)

	for _, t := range potentiallyQualifiedTableSyms {
		if 0 == len(t) {
			continue
		}

		if 1 != len(t) {
			return []string{}, errors.New(fmt.Sprintln("poorly qualified table:", t))
		}

		tableWhereClauses = append(tableWhereClauses, fmt.Sprintf("TABLE_NAME = '%s'", t[0]))
	}

	return []string{"(" + strings.Join(tableWhereClauses, " OR ") + ")"}, nil
}
