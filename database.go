// Package Query lets you build and execute SQL chainable queries against a database of your choice, and defer execution of SQL until you wish to extract a count or array of models.
package query

import (
	"database/sql"
	"fmt"
	"github.com/fragmenta/query/adapters"
	"time"
)

// At present we only ever have one database
// This should be fine as long as separate goroutines do not try to share a database

// Package global db  - this reference is not exported outside the package.
var database adapters.Database

// Open the database - builds a database adapter and assigns it to a package global query.database...
func OpenDatabase(opts map[string]string) error {

	// If we already have a db, return it
	if database != nil {
		return fmt.Errorf("Database already open - %s", database)
	}

	// Assign the db global in query package

	switch opts["adapter"] {
	case "sqlite3":
		database = &adapters.SqliteAdapter{}
	case "mysql":
		database = &adapters.MysqlAdapter{}
	case "postgres":
		database = &adapters.PostgresqlAdapter{}
	default:
		database = nil // fail
	}

	if database == nil {
		return fmt.Errorf("Database adapter not recognised - %s", opts)
	}

	// Ask the db adapter to open
	return database.Open(opts)
}

// Close the database opened by OpenDatabase
func CloseDatabase() error {
	var err error
	if database != nil {
		err = database.Close()
		database = nil
	}

	return err
}

// Execute the given sql Query against our database, with arbitrary args
func QuerySQL(query string, args ...interface{}) (*sql.Rows, error) {
	results, err := database.Query(query, args...)
	return results, err
}

// Execute the given sql against our database with arbitrary args
// NB returns sql.Result - not to be used when rows expected
func ExecSQL(query string, args ...interface{}) (sql.Result, error) {
	results, err := database.Exec(query, args...)
	return results, err
}

func TimeString(t time.Time) string {
	return database.TimeString(t)
}
