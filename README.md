Query
=====

Query lets you build SQL queries with chainable methods, and defer execution of SQL until you wish to extract a count or array of models. It will probably remain limited in scope - it is not intended to be a full ORM with strict mapping between db tables and structs, but a tool for querying the database with minimum friction, and performing CRUD operations linked to models; simplifying your use of SQL to store model data without getting in the way. Full or partial SQL queries are of course also available.

Supported databases: PostgreSQL, SQLite, MySQL. Bug fixes, suggestions and contributions welcome. 

Usage
=====


```go

// In your app - open a database with options
options := map[string]string{"adapter":"postgres","db":"query_test"}
err := query.OpenDatabase(options)
defer query.CloseDatabase()

...

// In your model
type Page struct {
	Id			int64
	CreatedAt   time.Time
	UpdatedAt   time.Time
	MyField	    myStruct
	...
	// Models can have any structure, any PK, here an int is used
}

...

// In your controller/action

// Start querying the database using chained finders
pages := page.Query().Where("id IN (?,?)",4,5).Order("id desc").Limit(30)

// Build up chains depending on other app logic, still no db requests
if shouldRestrict {
	pages.Where("id > ?",3).OrWhere("keywords ~* ?","Page")
}

// Pass the relation around, until you are ready to retrieve models from the db
for i,page range pages.Array() {...}
```

What it does
============

* Builds chainable queries including where, orwhere,group,having,order,limit,offset or plain sql
* Allows any Primary Key/Table name or model fields
* Allows Delete and Update operations on queried records, without creating objects
* Defers SQL query until full query is built and results requested
* Returns an array of models suitable for use with range, a single model, or a count
* Provides a simple CRUD interface for models to conform to, and helpers for common operations (CUD).
* Updates created at and updated at fields automatically on create and update if present
* Maps automatically from db_columns <-> StructFields, ignoring fields which do not exist


What it won't do
==================

* Require changes to your structs like tagging fields or adding an id
* Require certain model fields to be present in the db
* Cause problems with untagged fields, embedding, and fields not in the database
* Provide hooks after/before update etc - your models are in charge of queries
* Perform migrations


What it should do
==================

* Allow an optional map from db columns -> struct fields via a method on Model - at present reflection is used
* Handle Relations with common join patterns ( has_many, belongs_to etc)
* Possibly allow option to generate sql from db map for migrations - not sure this is in scope


Tests
==================

All 3 databases supported have a test suite - to run the tests, create a database called query_test in mysql and psql then run go test at the root of the package.

```bash
go test
```



Versions
==================

- 1.0 - First version with interfaces and chainable finders
- 1.0.1 - Updated to quote table names and fields, for use of reserved words, bug fix for mysql concurrency
