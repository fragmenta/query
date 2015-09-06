package adapters

import (
	"database/sql"
	"fmt"

	// Mysql driver
	//_ "github.com/Go-SQL-Driver/MySQL"
)

// Adapters conform to the query.Database interface
type MysqlAdapter struct {
	*Adapter
	options map[string]string
	sqlDB   *sql.DB
	debug   bool
}

// Open this database
func (db *MysqlAdapter) Open(opts map[string]string) error {

	db.debug = false
	db.options = map[string]string{
		"adapter":  "mysql",
		"user":     "root",
		"password": "",
		"db":       "query_test",
	}

	if opts["debug"] == "true" {
		db.debug = true
	}

	// Merge options
	for k, v := range opts {
		db.options[k] = v
	}

	//"user:password@/dbname?charset=utf8")
	options := fmt.Sprintf("%s:%s@/%s?charset=utf8", db.options["user"], db.options["password"], db.options["db"])
	var err error
	db.sqlDB, err = sql.Open(db.options["adapter"], options)
	if err != nil {
		return err
	}

	if db.sqlDB == nil {
		println("Mysql options:%s", options)
		return fmt.Errorf("\nError creating database with options:", db.options)
	}

	// Call ping on the db to check it does actually exist!
	err = db.sqlDB.Ping()
	if err != nil {
		return err
	}

	return err

}

// Close the database
func (db *MysqlAdapter) Close() error {
	if db.sqlDB != nil {
		return db.sqlDB.Close()
	}
	return nil
}

// Return the internal db.sqlDB pointer
func (db *MysqlAdapter) SqlDB() *sql.DB {
	return db.sqlDB
}

// Execute Query SQL - NB caller must call use defer rows.Close() with rows returned
func (db *MysqlAdapter) Query(query string, args ...interface{}) (*sql.Rows, error) {
	return db.performQuery(db.sqlDB, db.debug, query, args...)
}

// Exec - use this for non-select statements
func (db *MysqlAdapter) Exec(query string, args ...interface{}) (sql.Result, error) {
	return db.performExec(db.sqlDB, db.debug, query, args...)
}

// Quote a table name or column name
func (db *MysqlAdapter) QuoteField(name string) string {
	return fmt.Sprintf("`%s`", name)
}

// Insert a record with params and return the id - psql behaves differently
func (db *MysqlAdapter) Insert(query string, args ...interface{}) (id int64, err error) {

	tx, err := db.sqlDB.Begin()
	if err != nil {
		return 0, err
	}

	// Execute the sql using db
	result, err := db.Exec(query, args...)
	if err != nil {
		return 0, err
	}

	// TODO - check this works on mysql under load with concurrent connections
	// fine if connection not shared
	id, err = result.LastInsertId()
	if err != nil {
		return 0, err
	}

	err = tx.Commit()
	if err != nil {
		return 0, err
	}

	return id, nil

}
