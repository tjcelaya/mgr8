package mdb

import (
	"database/sql"
)

type ColumnDefinition struct {
	dbName, tableName, colName, colType string
	autoInc, nullable           bool
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
