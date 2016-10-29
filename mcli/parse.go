package mcli

import (
	"flag"
	_"github.com/tjcelaya/mgr8/mutil"
)

type MFlags struct {
	dbName,
	tableName,
	colName,
	typeName,
	newTypeName,
	user,
	pass *string
	maxDbConn *int
	forAutoIncAdd,
	forAutoIncRemoval,
	askForPass *bool
}

func parseFlags() MFlags {
	f := MFlags{
		dbName:            flag.String("db", "mgr8_db", "Database name to work on."),
		colName:           flag.String("column", "", "Column name usually 'id` or something like that."),
		tableName:         flag.String("table", "", "Table name"),
		typeName:          flag.String("coltype", "", "Column type"),
		newTypeName:       flag.String("newcoltype", "", "New Column type"),
		user:              flag.String("user", "root", "username"),
		pass:              flag.String("pass", "secret", "password in cleartext :("),
		askForPass:        flag.Bool("p", false, "password prompt"),
		maxDbConn:         flag.Int("connections", 5, "Number of connections to use, 0 means uses as many connections as tables"),
		forAutoIncAdd:     flag.Bool("add-auto-inc", false, "whether to add auto_increment"),
		forAutoIncRemoval: flag.Bool("remove-auto-inc", false, "whether to remove auto_increment"),
	}
	flag.Parse()

	// should we parse forAutoInc* into mutil.BinaryChangeIntent here?

	return f
}
