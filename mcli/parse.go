package mcli

import (
	"flag"
	_ "github.com/tjcelaya/mgr8/mutil"
	"strings"
)

type MFlags struct {
	verbose             *bool
	write               *bool
	dbName              *string
	tableName           *string
	colName             *string
	typeName            *string
	newTypeName         *string
	forAutoIncAdd       *bool
	forAutoIncRemoval   *bool
	characterSetName    *string
	newCharacterSetName *string
	collationName       *string
	newCollationName    *string
	user                *string
	pass                *string
	askForPass          *bool
	maxDbConn           *int
}

func parseFlags() MFlags {
	f := MFlags{
		verbose: flag.Bool("verbose", false, "Verbose?"),
		write:   flag.Bool("write", false, "Whether to actually perform changes"),

		dbName:    flag.String("db", "mgr8_db", "Database name to work on."),
		tableName: flag.String("table", "", "Table name"),
		colName:   flag.String("column", "", "Column name usually 'id' or something like that, can be comma-delimited OR include % wildcards, but not both."),

		typeName:    flag.String("coltype", "", "Column type"),
		newTypeName: flag.String("newcoltype", "", "New Column type"),

		forAutoIncAdd:     flag.Bool("add-auto-inc", false, "whether to add auto_increment"),
		forAutoIncRemoval: flag.Bool("remove-auto-inc", false, "whether to remove auto_increment"),

		characterSetName:    flag.String("character-set", "", "Character Set to match"),
		newCharacterSetName: flag.String("new-character-set", "", "Character Set to apply"),
		collationName:       flag.String("collation", "", "Collation to match"),
		newCollationName:    flag.String("new-collation", "", "Collation to match"),

		user:       flag.String("user", "root", "username"),
		pass:       flag.String("pass", "secret", "password in cleartext :("),
		askForPass: flag.Bool("p", false, "password prompt"),
		maxDbConn:  flag.Int("connections", 5, "Number of connections to use, 0 means uses as many connections as tables"),
	}
	flag.Parse()

	*f.tableName = strings.Trim(*f.tableName, `'"`)
	*f.colName = strings.Trim(*f.colName, `'"`)

	// should we parse forAutoInc* into mutil.BinaryChangeIntent here?

	return f
}
