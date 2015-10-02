package adapters

import (
	"database/sql"
	"fmt"

	// psql driver
	_ "github.com/lib/pq"
)

// PostgresqlAdapter conforms to the query.Database interface
type PostgresqlAdapter struct {
	*Adapter
	options map[string]string
	sqlDB   *sql.DB
	debug   bool
}

// Open this database
func (db *PostgresqlAdapter) Open(opts map[string]string) error {

	db.debug = false
	db.options = map[string]string{
		"adapter":  "postgres",
		"user":     "",
		"password": "",
		"db":       "",
		"sslmode":  "disable",
	}

	if opts["debug"] == "true" {
		db.debug = true
	}

	// Merge options
	for k, v := range opts {
		db.options[k] = v
	}

	// Default to psql database
	// TODO: add host= and port= options here, with default values of localhost:5432
	// which can be set from opts
	password := ""
	if len(db.options["password"]) > 0 {
		password = fmt.Sprintf("password=%s", db.options["password"])
	}
	optionString := fmt.Sprintf("user=%s %s dbname=%s sslmode=%s", db.options["user"], password, db.options["db"], db.options["sslmode"])
	var err error
	db.sqlDB, err = sql.Open(db.options["adapter"], optionString)
	if err != nil {
		return err
	}

	// Call ping on the db to check it does actually exist!
	err = db.sqlDB.Ping()
	if err != nil {
		return err
	}

	if db.sqlDB != nil && db.debug {
		fmt.Printf("Database %s opened using %s\n", db.options["db"], db.options["adapter"])
	}

	return nil

}

// Close the database
func (db *PostgresqlAdapter) Close() error {
	if db.sqlDB != nil {
		return db.sqlDB.Close()
	}
	return nil
}

// SQLDB returns the internal db.sqlDB pointer
func (db *PostgresqlAdapter) SQLDB() *sql.DB {
	return db.sqlDB
}

// Query executes query SQL - NB caller must call use defer rows.Close() with rows returned
func (db *PostgresqlAdapter) Query(query string, args ...interface{}) (*sql.Rows, error) {
	return db.performQuery(db.sqlDB, db.debug, query, args...)
}

// Exec - use this for non-select statements
func (db *PostgresqlAdapter) Exec(query string, args ...interface{}) (sql.Result, error) {
	return db.performExec(db.sqlDB, db.debug, query, args...)
}

// Placeholder returns the db placeholder
func (db *PostgresqlAdapter) Placeholder(i int) string {
	return fmt.Sprintf("$%d", i)
}

// InsertSQL is extra SQL for end of insert statement (RETURNING for psql)
func (db *PostgresqlAdapter) InsertSQL(pk string) string {
	return fmt.Sprintf("RETURNING %s", pk)
}

// Insert a record with params and return the id
func (db *PostgresqlAdapter) Insert(sql string, args ...interface{}) (id int64, err error) {

	// TODO - handle different types of id, not just int
	// Execute the sql using db and retrieve new row id
	row := db.sqlDB.QueryRow(sql, args...)
	err = row.Scan(&id)
	return id, err
}
