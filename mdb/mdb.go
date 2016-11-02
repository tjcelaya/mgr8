package mdb

import (
	"database/sql"
)

type ColumnDefinition struct {
	dbName, tableName, colName, colType, extra string
	autoInc, nullable, unsigned           bool
}

func NewColumnDefinition(
	dbName, tableName, colName, colType, extra string,
	autoInc, nullable, unsigned           bool) ColumnDefinition {
	return ColumnDefinition{dbName, tableName, colName, colType, extra, autoInc, nullable, unsigned}
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

type AlterError string