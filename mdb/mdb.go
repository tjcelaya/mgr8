package mdb

import (
	"database/sql"
	"strings"
)

type AlterError string

type ColumnDefinition struct {
	dbName, tableName, colName, colType, characterSetName, collationName, defaultValue, extra string
	autoInc, isNullable, isUnsigned, hasDefault bool
}

func NewColumnDefinition(
	dbName, tableName, colName, colType, characterSetName, collationName, defaultValue, extra string,
	autoInc, isNullable, isUnsigned, hasDefault bool) ColumnDefinition {
	return ColumnDefinition{
		dbName,
		tableName,
		colName,
		colType,
		characterSetName,
		collationName,
		defaultValue,
		extra,
		autoInc,
		isNullable,
		isUnsigned,
		hasDefault}
}

func New(dsn string, maxConn int) (db *sql.DB, err error) {
	db, err = sql.Open("mysql", dsn)

	if err != nil {
		return nil, err
	}

	err = db.Ping()

	if err != nil {
		return nil, err
	}

	db.SetMaxOpenConns(maxConn)
	db.SetMaxIdleConns(maxConn)

	return db, nil
}

func buildColumnDefinitionQuery() string {
	return `
        SELECT
            TABLE_SCHEMA dbName,
            TABLE_NAME tableName,
            COLUMN_NAME colName,
            COLUMN_TYPE colType,
            IS_NULLABLE nullable,
            IF(ISNULL(CHARACTER_SET_NAME),
                '',
                CONCAT(' CHARACTER SET ', CHARACTER_SET_NAME)) characterSetName,
            IF(ISNULL(COLLATION_NAME),
                '',
                CONCAT(' COLLATE ', COLLATION_NAME)) collationName,
            COLUMN_DEFAULT IS NOT NULL hasDefault,
            QUOTE(COLUMN_DEFAULT) defaultValue,
            EXTRA extra
        FROM
            information_schema.COLUMNS
        WHERE `
}

func hydrateColumnDefinitions(rows *sql.Rows) (colDefs []ColumnDefinition, err error) {
	for rows.Next() {
		var colDef ColumnDefinition
		var extraStr, nullableStr string
		err = rows.Scan(
			&colDef.dbName,
			&colDef.tableName,
			&colDef.colName,
			&colDef.colType,
			&nullableStr,
			&colDef.characterSetName,
			&colDef.collationName,
			&colDef.hasDefault,
			&colDef.defaultValue,
			&extraStr)

		colDef.autoInc = strings.Contains(extraStr, "auto_increment")
		colDef.isUnsigned = strings.Contains(colDef.colType, "unsigned")
		colDef.isNullable = nullableStr == "YES"

		if err != nil {
			return colDefs, err
		}

		colDefs = append(colDefs, colDef)
	}

	return colDefs, nil
}

func isStringType(typeName string) bool {
	return strings.Contains(typeName, "CHAR") || strings.Contains(typeName, "TEXT")
}