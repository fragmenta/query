package adapters

import (
	"database/sql"
	"fmt"

	// psql driver
	_ "github.com/lib/pq"
)

// Adapters conform to the query.Database interface
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
	options_string := fmt.Sprintf("user=%s password=%s dbname=%s sslmode=%s", db.options["user"], db.options["password"], db.options["db"], db.options["sslmode"])
	var err error
	db.sqlDB, err = sql.Open(db.options["adapter"], options_string)
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

// Return the internal db.sqlDB pointer
func (db *PostgresqlAdapter) SqlDB() *sql.DB {
	return db.sqlDB
}

// Execute Query SQL - NB caller must call use defer rows.Close() with rows returned
func (db *PostgresqlAdapter) Query(query string, args ...interface{}) (*sql.Rows, error) {
	return db.performQuery(db.sqlDB, db.debug, query, args...)
}

// Exec - use this for non-select statements
func (db *PostgresqlAdapter) Exec(query string, args ...interface{}) (sql.Result, error) {
	return db.performExec(db.sqlDB, db.debug, query, args...)
}

func (db *PostgresqlAdapter) Placeholder(i int) string {
	return fmt.Sprintf("$%d", i)
}

// Extra SQL for end of insert statement (RETURNING for psql)
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
