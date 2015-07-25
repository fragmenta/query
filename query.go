// Package Query lets you build and execute SQL chainable queries against a database of your choice, and defer execution of SQL until you wish to extract a count or array of models.

// TODO  // Decide on the best interface:

// var pages []*Page
// err := q.Fetch(&pages)
// OR 
// pages, err := PagesResults(q)
// I think I prefer fetch & because then it is clear what type you have
// if so then FetchFirst() instead of FirstResult?
// We should pick one of these options and go with it...


// NB in order to allow cross-compilation, we exlude sqlite drivers by default
// uncomment them to allow use of sqlite


package query

import (
	"database/sql"
	"errors"
	"fmt"
	"reflect"
	"sort"
	"strings"
)

// query.Debug sets whether we output debug statements for SQL
var Debug bool

func init() {
	Debug = false // default to false
}

// The results of a query are returned as map[string]interface{} by default
type Result map[string]interface{}

// A function which applies effects to queries
type QueryFunc func(q *Query) *Query

// Struct relation provides all the chainable relational query builder methods
type Query struct {

	// Database - database name and primary key, set with New()
	tablename  string
	primarykey string

	// SQL - Private fields used to store sql before building sql query
	sql    string
	sel    string
	join   string
	where  string
	group  string
	having string
	order  string
	offset string
	limit  string

	// Extra args to be substituted in the *where* clause
	args []interface{}
}

// Build a new Query, given the table and primary key
func New(t string, pk string) *Query {

	// If we have no db, return nil
	if database == nil {
		return nil
	}

	q := &Query{
		tablename:  t,
		primarykey: pk,
	}

	return q
}

// Execute the given sql and args against the database directly
// Returning sql.Result (NB not rows)
func Exec(sql string, args ...interface{}) (sql.Result, error) {
	results, err := database.Exec(sql, args...)
	return results, err
}


// Execute the given sql and args against the database directly
// Returning sql.Rows
func Rows(sql string, args ...interface{}) (*sql.Rows, error) {
	results, err := database.Query(sql, args...)
	return results, err
}


// These should instead be something like query.New("table_name").Join(a,b).Insert() and just have one multiple function?
func (q *Query) InsertJoin(a int64, b int64) error {
	return q.InsertJoins([]int64{a},[]int64{b})
}

// Insert joins using an array of ids (more general version of above)
// This inserts joins for every possible relation between the ids
func (q *Query) InsertJoins(a []int64, b []int64) error {

	// Make sure we have some data
	if len(a) == 0 || len(b) == 0 {
		return errors.New(fmt.Sprintf("Null data for joins insert %s", q.table()))
	}

    // Check for null entries in start of data - this is not a good idea. 
//	if a[0] == 0 || b[0]  == 0 {
//		return errors.New(fmt.Sprintf("Zero data for joins insert %s", q.table()))
//	}

    
	values := ""
	for _, av := range a {
        for _, bv := range b {
            // NB no zero values allowed, we simply ignore zero values
            if av != 0 && bv != 0 {
                values += fmt.Sprintf("(%d,%d),", av, bv) 
            }
           
        }
	}
            

	values = strings.TrimRight(values, ",")

	sql := fmt.Sprintf("INSERT into %s VALUES %s;", q.table(), values)


	if Debug {
		fmt.Printf("JOINS SQL:%s\n", sql)
	}
    

	_, err := database.Exec(sql)
	return err
}


// Update the given joins, using the given id to clear joins first
func (q *Query) UpdateJoins(id int64, a []int64, b []int64) error {
    
	if Debug {
		fmt.Printf("SetJoins %s %s=%d: %v %v \n",q.table(),q.pk(), id,a,b)
	}
    
	// First delete any existing joins
 	err := q.Where(fmt.Sprintf("%s=?",q.pk()), id).Delete()
	if err != nil {
		return err
	}
    
    // Now join all a's with all b's by generating joins for each possible combination

	// Make sure we have data in both cases, otherwise do not attempt insert any joins
	if len(a) > 0 && len(b) > 0 {
    	// Now insert all new ids - NB the order of arguments here MUST match the order in the table
    	err = q.InsertJoins(a, b)
    	if err != nil {
    		return err
    	}
	}
    
	return nil
}

// SetJoins above is based on a faulty assumption - that if we have 1 in an array that's the one to use for id. 
// FALSE we might have 1 in both arrays




func (q *Query) Insert(params map[string]string) (int64, error) {

	// Insert and retreive ID in one step from db
	sql := q.insertSQL(params)

	if Debug {
		fmt.Printf("INSERT SQL:%s %v\n", sql, valuesFromParams(params))
	}

	id, err := database.Insert(sql, valuesFromParams(params)...)
	if err != nil {
		return 0, err
	}

	return id, nil
}

// Used for update statements, turn params into sql i.e. "col"=?
// NB we always use parameterized queries, never string values.
func (q *Query) insertSQL(params map[string]string) string {
	cols := make([]string, 0)
	vals := make([]string, 0)
	for i, k := range sortedParamKeys(params) {
		cols = append(cols, database.QuoteField(k))
		vals = append(vals, database.Placeholder(i+1))
	}
	query := fmt.Sprintf("INSERT INTO %s (%s) VALUES(%s) %s;", q.table(), strings.Join(cols, ","), strings.Join(vals, ","), database.InsertSQL(q.pk()))

	return query
}

// Update one model specified in this query - the column names MUST be verified in the model
func (q *Query) Update(params map[string]string) error {
	// We should check the query has a where limitation to avoid updating all?
	// pq unfortunately does not accept limit(1) here
	return q.UpdateAll(params)
}

// Delete one model specified in this relation
func (q *Query) Delete() error {
	// We should check the query has a where limitation?
	return q.DeleteAll()
}

// Update all models specified in this relation
func (q *Query) UpdateAll(params map[string]string) error {
	// Create sql for update from ALL params
	q.Select(fmt.Sprintf("UPDATE %s SET %s", q.table(), querySQL(params)))

	// Execute, after PREpending params to args
	// in an update statement, the where comes at the end
	q.args = append(valuesFromParams(params), q.args...)

	if Debug {
		fmt.Printf("UPDATE SQL:%s\n%v\n", q.QueryString(), valuesFromParams(params))
	}

	_, err := q.Result()

	return err
}

// Delete all models specified in this relation
func (q *Query) DeleteAll() error {

    
	q.Select(fmt.Sprintf("DELETE FROM %s", q.table()))
	
    if Debug {
		fmt.Printf("DELETE SQL:%s <= %v\n", q.QueryString(), q.args)
	}
    
	// Execute
	_, err := q.Result()

	return err
}

// Fetch a count of model objects (executes SQL).
func (q *Query) Count() (int64, error) {

	// In order to get consistent results, we use the same query builder
	// but reset select to simple count select
	s := q.sel
	o := strings.Replace(q.sql, "ORDER BY ", "", 1)// FIXME - is this mistaken? Should it in fact be o := q.order?
	q.order = "" // Order must be blank on count
	countSelect := fmt.Sprintf("SELECT COUNT(%s) FROM %s", q.pk(), q.table())
    q.Select(countSelect)
    

	// Fetch count from db for our sql
	var count int64 = 0
	rows, err := q.Rows()

	if err != nil {
		return 0, fmt.Errorf("Error querying database for count: %s\nQuery:%s", err, q.QueryString())

	} else {
		defer rows.Close()
		// We expect just one row, with one column (count)
		for rows.Next() {
			err := rows.Scan(&count)
			if err != nil {
				return 0, err
			}
		}

	}
    
	// Reset select after getting count query
	q.Select(s)
	q.Order(o)
	q.reset()

	return count, err
}


// Execute the query against the database, returning sql.Result, and error (no rows)
// (Executes SQL)
func (q *Query) Result() (sql.Result, error) {
	results, err := database.Exec(q.QueryString(), q.args...)
	return results, err
}


// Execute the query against the database, and return the sql rows result for this query
// (Executes SQL)
func (q *Query) Rows() (*sql.Rows, error) {
	results, err := database.Query(q.QueryString(), q.args...)
	return results, err
}




// Return the next row of results, consumable by a model - the columns returned depend on the sql select
func (q *Query) FirstResult() (Result, error) {

	// Set a limit on the query
	q.Limit(1)

	// Fetch all results (1)
	results, err := q.Results()
	if err != nil {
		return nil, err
	}

	if len(results) == 0 {
		return nil, errors.New(fmt.Sprintf("No results found for Query:%s", q.QueryString()))
	}

	// Return the first result
	return results[0], nil
}

// Return the next row of results, consumable by a model - the columns returned depend on the sql select
// this allows handling returns of partial data or joined data without scanning into custom structs
func (q *Query) Results() ([]Result, error) {

	// Make an empty result set map
	results := make([]Result, 0)

	// Fetch rows from db for our sql
	rows, err := q.Rows()

	if err != nil {
		return results, fmt.Errorf("Error querying database for rows: %s\nQUERY:%s", err, q)
	} else {
		// Close rows before returning
		defer rows.Close()

		// Fetch the columns from the database
		cols, err := rows.Columns()
		if err != nil {
			return results, fmt.Errorf("Error fetching columns: %s\nQUERY:%s\nCOLS:%s", err, q, cols)
		}

		// For each row, construct an entry in results with a map of column string keys to values
		for rows.Next() {
			result, err := scanRow(cols, rows)
			if err != nil {
				return results, fmt.Errorf("Error fetching row: %s\nQUERY:%s\nCOLS:%s", err, q, cols)
			}
			results = append(results, result)
		}

	}

	return results, nil
}

func (q *Query) ResultIds() []int64 {
    var ids []int64
	if Debug {
		fmt.Printf("#info ResultIds:%s\n", q.DebugString())
    }
    results, err := q.Results()
    if err != nil {
        return ids
    }
    
    for _,r := range results {
        if r["id"] != nil {
            ids = append(ids,r["id"].(int64))   
        }
    }

    return ids
}



// I'm in two minds about this - it is neater in that we init with strings
// but uglier in that we have to 
// Can we also assign a model which conforms to an interface for - 
// New() to create each new record
// Array() to create the array which we append the records to?
// Avoid using reflect here if we possibly can

// FIXME - reasses whether we want to provide this interface before release?

// Fetch an array of model objects (Executes SQL)
// accepts a pointer to an array of model pointers
func (q *Query) Fetch(sliceptr interface{}) error {
	
    
	// Check for errors with sliceptr param - we need a ptr to a slice
	spv := reflect.ValueOf(sliceptr)
	if spv.Kind() != reflect.Ptr || spv.IsNil() {
		return errors.New("Valid slice pointer required")
	}

	// Fetch rows from db for our sql
	rows, err := q.Rows()

	if err != nil {
		return fmt.Errorf("Error querying database for rows: %s\nQUERY:%s", err, q.QueryString())
	} else {

		defer rows.Close()
		// We iterate over the rows and pass the column values to the model to update it
		cols, err := rows.Columns()
		if err != nil {
			return fmt.Errorf("Error fetching columns: %s\nQUERY:%s\nCOLS:%s", err, q.QueryString(), cols)
		}

		// Use of reflection unavoidable here without ugly casts from caller
		// This is nasty and I'd prefer to remove it, but appears to be required by Golang's type system
		// Unless we pass in the New func or model...

		mt := spv.Type().Elem().Elem().Elem()

		models := spv.Elem()

		for rows.Next() {
			result, err := scanRow(cols, rows)
			if err != nil || result == nil {
				return fmt.Errorf("Error scanning row: %s", err)
			}

			vp := reflect.New(mt)
			mf := vp.MethodByName("New")
			if mf != *new(reflect.Value) {
				// Call New on model *if* the function exists - need to check signature too...
				mv := mf.Call([]reflect.Value{reflect.ValueOf(result)})
				models.Set(reflect.Append(models, mv[0]))
			} else {
				models.Set(reflect.Append(models, vp))
			}

		}

	}

	return nil
}






// Build a query string to use for results
func (q *Query) QueryString() string {

	if q.sql == "" {

		// if we have arguments override the selector
		if q.sel == "" {
			// Note q.table() etc perform quoting on field names
			q.sel = fmt.Sprintf("SELECT %s.* FROM %s", q.table(), q.table())
		}

		q.sql = fmt.Sprintf("%s %s %s %s %s %s %s %s", q.sel, q.join, q.where, q.group, q.having, q.order, q.offset, q.limit)
		q.sql = strings.TrimRight(q.sql, " ")
		q.sql = strings.Replace(q.sql, "  ", " ", -1)
		q.sql = strings.Replace(q.sql, "   ", " ", -1)

		// Replace ? with whatever placeholder db prefers
		q.replaceArgPlaceholders()

		q.sql = q.sql + ";"
	}

	return q.sql
}

// CHAINABLE FINDERS

// Apply the QueryFunc to this query, and return the modified Query
// This allows chainable finders from other packages
// e.g. q.Apply(status.Published) where status.Published is a QueryFunc
func (q *Query) Apply(f QueryFunc) *Query {
	return f(q)
}

// Apply the conditions to this - this allows conditions to be applied from other packages
// e.g. q.Conditions(role.Owner,status.Published)
func (q *Query) Conditions(funcs ...QueryFunc) *Query {
	for _, f := range funcs {
		q = f(q)
	}
	return q
}

// Define all sql together - overrides all other setters
func (q *Query) Sql(sql string) *Query {
	q.sql = sql // Completely replace all stored sql
	q.reset()
	return q
}

// Define limit with an int
func (q *Query) Limit(limit int) *Query {
	q.limit = fmt.Sprintf("LIMIT %d", limit)
	q.reset()
	return q
}

// Define limit with an int
func (q *Query) Offset(offset int) *Query {
	q.offset = fmt.Sprintf("OFFSET %d", offset)
	q.reset()
	return q
}

// Define where clause on SQL - Additional calls add WHERE () AND () clauses
func (q *Query) Where(sql string, args ...interface{}) *Query {

	if len(q.where) > 0 {
		q.where = fmt.Sprintf("%s AND (%s)", q.where, sql)
	} else {
		q.where = fmt.Sprintf("WHERE (%s)", sql)
	}

	// NB this assumes that args are only supplied for where clauses
	// this may be an incorrect assumption!
	if args != nil {
		if q.args == nil {
			q.args = args
		} else {
			q.args = append(q.args, args...)
		}
	}

	q.reset()
	return q
}

// Define where clause on SQL - Additional calls add WHERE () OR () clauses
func (q *Query) OrWhere(sql string, args ...interface{}) *Query {

	if len(q.where) > 0 {
		q.where = fmt.Sprintf("%s OR (%s)", q.where, sql)
	} else {
		q.where = fmt.Sprintf("WHERE (%s)", sql)
	}

	if args != nil {
		if q.args == nil {
			q.args = args
		} else {
			q.args = append(q.args, args...)
		}
	}

	q.reset()
	return q
}

// Define a join clause on SQL - we create an inner join like this:
// INNER JOIN extras_seasons ON extras.id = extra_id
// q.Select("SELECT units.* FROM units INNER JOIN sites ON units.site_id = sites.id")

// rails join example
// INNER JOIN "posts_tags" ON "posts_tags"."tag_id" = "tags"."id" WHERE "posts_tags"."post_id" = 111

func (q *Query) Join(otherModel string) *Query {
	modelTable := q.tablename

	tables := []string{
		modelTable,
		ToPlural(otherModel),
	}
	sort.Strings(tables)
	joinTable := fmt.Sprintf("%s_%s", tables[0], tables[1])

	sql := fmt.Sprintf("INNER JOIN %s ON %s.id = %s.%s_id", database.QuoteField(joinTable), database.QuoteField(modelTable), database.QuoteField(joinTable), ToSingular(modelTable))

	if len(q.join) > 0 {
		q.join = fmt.Sprintf("%s %s", q.join, sql)
	} else {
		q.join = fmt.Sprintf("%s", sql)
	}

	q.reset()
	return q
}

// Define order sql
func (q *Query) Order(sql string) *Query {
	if sql == "" {
		q.order = ""
	} else {
		q.order = fmt.Sprintf("ORDER BY %s", sql)
	}
	q.reset()

	return q
}

// Define group by sql
func (q *Query) Group(sql string) *Query {
	if sql == "" {
		q.group = ""
	} else {
		q.group = fmt.Sprintf("GROUP BY %s", sql)
	}
	q.reset()
	return q
}

// Define having sql
func (q *Query) Having(sql string) *Query {
	if sql == "" {
		q.having = ""
	} else {
		q.having = fmt.Sprintf("HAVING %s", sql)
	}
	q.reset()
	return q
}

// Define select sql
func (q *Query) Select(sql string) *Query {
	q.sel = sql
	q.reset()
	return q
}



// Return a debug string with current query + args (for debugging)
func (q *Query) DebugString() string {
	return fmt.Sprintf("--\nQuery-SQL:%s\nARGS:%s\n--", q.QueryString(), q.argString())
}

// Clear sql/query caches
func (q *Query) reset() {
	// Perhaps later clear cached compiled representation of query too

	// clear stored sql
	q.sql = ""
}

// Return an arg string (for debugging)
func (q *Query) argString() string {
	output := "-"

	for _, a := range q.args {
		output = output + fmt.Sprintf("'%s',", q.argToString(a))
	}
	output = strings.TrimRight(output, ",")
	output = output + ""

	return output
}

// Convert arguments to string - used only for debug argument strings
// Not to be exported or used to try to escape strings...
func (q *Query) argToString(arg interface{}) string {
	switch arg.(type) {
	case string:
		return arg.(string)
	case []byte:
		return string(arg.([]byte))
	case int, int8, int16, int32, uint, uint8, uint16, uint32:
		return fmt.Sprintf("%d", arg)
	case int64, uint64:
		return fmt.Sprintf("%d", arg)
	case float32, float64:
		return fmt.Sprintf("%f", arg)
    case bool:
		return fmt.Sprintf("%d", arg)
    default:
		return fmt.Sprintf("%v", arg)
 
	}

	return ""
}

// Ask model for primary key name to use
func (q *Query) pk() string {
	return database.QuoteField(q.primarykey)
}

// Ask model for table name to use
func (q *Query) table() string {
	return database.QuoteField(q.tablename)
}

// Replace ? with whatever database prefers (psql uses numbered args)
func (q *Query) replaceArgPlaceholders() {
	// Match ? and replace with argument placeholder from database
	for i, _ := range q.args {
		q.sql = strings.Replace(q.sql, "?", database.Placeholder(i+1), 1)
	}
}

// Sorts the param names given - map iteration order is explicitly random in Go
// but we need params in a defined order to avoid unexpected results.
func sortedParamKeys(params map[string]string) []string {
	sortedKeys := make([]string, len(params))
	i := 0
	for k, _ := range params {
		sortedKeys[i] = k
		i++
	}
	sort.Strings(sortedKeys)

	return sortedKeys
}

// Generate a set of values for the params in order
func valuesFromParams(params map[string]string) []interface{} {

	// NB DO NOT DEPEND ON PARAMS ORDER - see note on SortedParamKeys
	values := make([]interface{}, 0)
	for _, key := range sortedParamKeys(params) {
		values = append(values, params[key])
	}
	return values
}

// Used for update statements, turn params into sql i.e. "col"=?
func querySQL(params map[string]string) string {
	output := make([]string, 0)
	for _, k := range sortedParamKeys(params) {
		output = append(output, fmt.Sprintf("%s=?", database.QuoteField(k)))
	}
	return strings.Join(output, ",")
}



func scanRow(cols []string, rows *sql.Rows) (Result, error) {

	// We return a map[string]interface{} for each row scanned
	result := Result{}

	values := make([]interface{}, len(cols))
	for i := 0; i < len(cols); i++ {
		var col interface{}
		values[i] = &col
	}

	// Scan results into these interfaces
	err := rows.Scan(values...)
	if err != nil {
		return nil, fmt.Errorf("Error scanning row: %s", err)
	}

	// Make a string => interface map and hand off to caller
	for i := 0; i < len(cols); i++ {
		value := reflect.Indirect(reflect.ValueOf(values[i]))
		v := *values[i].(*interface{})
		if value.Interface() != nil {
			switch v.(type) {
			default:
				result[cols[i]] = v
			case bool:
				result[cols[i]] = v.(bool)
			case int:
				result[cols[i]] = int64(v.(int))
			case []byte: // text cols are given as bytes
				result[cols[i]] = string(v.([]byte))
			case int64:
				result[cols[i]] = v.(int64)
			}
		}

	}

	return result, nil
}
