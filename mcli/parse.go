package mcli

import "flag"

type MFlags struct {
	dbName, colName, user, pass *string
	maxDbConn                   *int
	forAdd, forRemoval          *bool
}

func parseFlags() MFlags {
	f := MFlags{
		dbName:     flag.String("db", "db", "Database name to work on."),
		colName:    flag.String("column", "id", "Column name usually 'id'."),
		user:       flag.String("user", "root", "username"),
		pass:       flag.String("pass", "secret", "password"),
		maxDbConn:  flag.Int("connections", 5, "Number of connections to use, 0 means uses as many connections as tables"),
		forAdd:     flag.Bool("add-auto-inc", false, "whether to add auto_increment"),
		forRemoval: flag.Bool("remove-auto-inc", false, "whether to remove auto_increment"),
	}
	flag.Parse()
	return f
}
