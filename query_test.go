package query

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"testing"
	"time"
)

// FIXME - tests for this package are currently broken and use the old scheme

// Enable db debug mode to enable query logging
var debug string = "false"

var Format string = "\n---\nFAILURE\n---\ninput:    %q\nexpected: %q\noutput:   %q"

type Test struct {
	input    string
	expected string
}

func testLog() {
	pc, _, line, _ := runtime.Caller(1)
	name := runtime.FuncForPC(pc).Name()
	fmt.Printf("TEST LOG line %d - %s\n", line, name)
}

// AN EXAMPLE MODEL FOR TESTING - MATCHES THE TEST DB in some respects but not in others

// NB this model does not reflect normal usage as it is inside the query package, so methods have been renamed.
type Page struct {
	Id        int64
	UpdatedAt time.Time
	CreatedAt time.Time

	OtherField map[string]string // unused
	Title      string
	Summary    string
	Text       string

	UnusedField int8 // unused

}

func (p *Page) New(cols map[string]interface{}) *Page {
	page := &Page{}
	page.Id = cols["id"].(int64)

	// Normally we'd use validate.Int() etc for this - should those methods be built into fragmenta base model?

	// Deal with sqlite dates stored as strings
	// Normally times are times
	if _, ok := cols["created_at"].(time.Time); ok {
		page.CreatedAt = cols["created_at"].(time.Time)
	} else if cols["created_at"] != nil {
		page.CreatedAt, _ = time.Parse("2006-01-02 15:04:05.999 +0000", cols["created_at"].(string))
	}

	if _, ok := cols["updated_at"].(time.Time); ok {
		page.UpdatedAt = cols["updated_at"].(time.Time)
	} else if cols["updated_at"] != nil {
		page.UpdatedAt, _ = time.Parse("2006-01-02 15:04:05.999 +0000", cols["updated_at"].(string))
	}

	if cols["summary"] != nil {
		page.Summary = cols["summary"].(string)
	}
	if cols["title"] != nil {
		page.Title = cols["title"].(string)
	}
	if cols["text"] != nil {
		page.Text = cols["text"].(string)
	}

	page.UnusedField = 99 // unused

	return page

}

// Create a new query relation referencing this model
func (p *Page) Query() *Query {
	return New("pages", "id")
}

// Create a model object, called from actions.
func (p *Page) Create(params map[string]string) (int64, error) {
	// Update/add some params by default
	params["created_at"] = TimeString(time.Now().UTC())
	params["updated_at"] = TimeString(time.Now().UTC())

	return p.Query().Insert(params)
}

// Update this model object, called from actions.
func (p *Page) Update(params map[string]string) error {

	params["updated_at"] = TimeString(time.Now().UTC())

	return p.Query().Where("id=?", p.Id).Update(params)
}

// Delete this page
func (p *Page) Delete() error {
	return p.Query().Where("id=?", p.Id).Delete()
}

func (p *Page) String() string {
	return fmt.Sprintf("%d-%s-Last updated:%s", p.Id, p.Title, p.UpdatedAt)
}

// Fetch all results for this query
func PageFind(id int64) (*Page, error) {
	result, err := PageQuery().Where("id=?", id).FirstResult()
	if err != nil {
		return nil, err
	}
	return (&Page{}).New(result), nil
}

// These would be in a pages package and named First and All, so pages.All(q), and pages.First(q)

// Fetch the first result for this query
func PagesFirst(q *Query) (*Page, error) {

	result, err := q.FirstResult()
	if err != nil {
		return nil, err
	}
	return (&Page{}).New(result), nil
}

// Fetch all results for this query
func PagesResults(q *Query) ([]*Page, error) {

	results, err := q.Results()
	if err != nil {
		return nil, err
	}

	pages := make([]*Page, 0)
	for _, r := range results {
		pages = append(pages, (&Page{}).New(r))
	}

	return pages, nil
}

// A convenience method for testing - return a page query
func PageQuery() *Query {
	return New("pages", "id")
}

// ----------------------------------
// SQLITE TESTS
// ----------------------------------

func TestSQSetup(t *testing.T) {

	fmt.Println("\n---\nTESTING SQLITE\n---")

	// NB we use binary named sqlite3 - this is the default on OS X
	// First execute sql (NB test might not work on windows)
	// NB this requires sqlite3 version > 3.7.15 for init alternative would be to echo sql file at end
	cmd := exec.Command("sqlite3", "--init", "tests/query_test_sqlite.sql", "tests/query_test.sqlite")
	stdout, err := cmd.StdoutPipe()
	stderr, err := cmd.StderrPipe()
	err = cmd.Start()
	if err != nil {
		fmt.Println("Could not set up sqlite db - ERROR ", err)
		os.Exit(1)
	}
	go io.Copy(os.Stdout, stdout)
	go io.Copy(os.Stderr, stderr)
	cmd.Wait()

	if err == nil {
		_ = strings.Replace("", "", "", -1)

		// Open the database
		options := map[string]string{
			"adapter": "sqlite3",
			"db":      "tests/query_test.sqlite",
		}

		// for more detail on failure, enable debug mode on db
		options["debug"] = debug

		err = OpenDatabase(options)
		if err != nil {
			fmt.Println("Open database ERROR ", err)
			os.Exit(1)
		}

		fmt.Println("---\nQuery Testing Sqlite3 - DB setup complete\n---")
	}

}

func TestSQFind(t *testing.T) {

	// This should work - NB in normal usage this would be query.New
	p, err := PageFind(1)
	if err != nil {
		t.Fatalf(Format, "Find(1)", "Model object", err)
	}
	// Check we got the page we expect
	if p.Id != 1 {
		t.Fatalf(Format, "Find(1) p", "Model object", p)
	}

	// This should fail, so we check that
	p, err = PageFind(11)
	if err == nil || p != nil {
		t.Fatalf(Format, "Find(11)", "Model object", err)
	}

}

func TestSQCount(t *testing.T) {

	// This should return 3
	count, err := PageQuery().Count()
	if err != nil || count != 3 {
		t.Fatalf(Format, "Count failed", "3", fmt.Sprintf("%d", count))
	}

	// This should return 2 - test limit ignored
	count, err = PageQuery().Where("id in (?,?)", 1, 2).Order("id desc").Limit(100).Count()
	if err != nil || count != 2 {
		t.Fatalf(Format, "Count id < 3 failed", "2", fmt.Sprintf("%d", count))
	}

	// This should return 0
	count, err = PageQuery().Where("id > 3").Count()
	if err != nil || count != 0 {
		t.Fatalf(Format, "Count id > 3 failed", "0", fmt.Sprintf("%d", count))
	}

	// Test retrieving an array, then counting, then where
	// This should work
	q := PageQuery().Where("id > ?", 1).Order("id desc")

	count, err = q.Count()
	if err != nil || count != 2 {
		t.Fatalf(Format, "Count id > 1 failed", "2", fmt.Sprintf("%d", count), err)
	}

	// Reuse same query to get array after count
	results, err := q.Results()
	if err != nil || len(results) != 2 {
		t.Fatalf(Format, "Where Array after count", "len 2", err)
	}

}

func TestSQWhere(t *testing.T) {

	q := PageQuery().Where("id > ?", 1)

	pages, err := PagesResults(q)

	if err != nil || len(pages) != 2 {
		t.Fatalf(Format, "Where Array", "len 2", fmt.Sprintf("%d", len(pages)))
	}

}

func TestSQOrder(t *testing.T) {

	// Look for pages in reverse order
	var models []*Page
	q := PageQuery().Where("id > 0").Order("id desc")
	err := q.Fetch(&models)


	if err != nil || len(models) == 0 {
		t.Fatalf(Format, "Order count test id desc", "3", fmt.Sprintf("%d", len(models)))
		return
	}

	p := models[0]
	if p.Id != 3 {
		t.Fatalf(Format, "Order test id desc 1", "3", fmt.Sprintf("%d", p.Id))
		return
	}

	// Look for pages in right order - reset models
	models = make([]*Page, 0)
	q = PageQuery().Where("id < ?", 10).Where("id < ?", 100).Order("id asc")
	err = q.Fetch(&models)
	//   fmt.Println("TESTING MODELS %v",models)

	if err != nil || models == nil {
		t.Fatalf(Format, "Order test id asc count", "1", err)
	}

	p = models[0]
	if p.Id != 1 {
		t.Fatalf(Format, "Order test id asc 1", "1", fmt.Sprintf("%d", p.Id))
		return
	}

}

func TestSQSelect(t *testing.T) {

	var models []*Page
	q := PageQuery().Select("SELECT id,title from pages").Order("id asc")
	err := q.Fetch(&models)
	if err != nil || len(models) == 0 {
		t.Fatalf(Format, "Select error on id,title", "id,title", err)
	}
	p := models[0]
	if p.Id != 1 || p.Title != "Title 1." || len(p.Text) > 0 {
		t.Fatalf(Format, "Select id,title", "id,title only", p)
	}

}

func TestSQUpdate(t *testing.T) {

	p, err := PageFind(3)
	if err != nil {
		t.Fatalf(Format, "Update could not find model err", "id-3", err)
	}

	// Should really test updates with several strings here
	err = p.Update(map[string]string{"title": "UPDATE 1", "summary": "Test summary"})

	// Check it is modified
	p, err = PageFind(3)

	if err != nil {
		t.Fatalf(Format, "Error after update 1", "updated", err)
	}

	if p.Title != "UPDATE 1" {
		t.Fatalf(Format, "Error after update 1 - Not updated properly", "UPDATE 1", p.Title)
	}

}

// Some more damaging operations we execute at the end,
// to avoid having to reload the db for each test

func TestSQUpdateAll(t *testing.T) {

	err := PageQuery().UpdateAll(map[string]string{"title": "test me"})
	if err != nil {
		t.Fatalf(Format, "UPDATE ALL err", "udpate all records", err)
	}

	// Check we have all pages with same title
	count, err := PageQuery().Where("title=?", "test me").Count()

	if err != nil || count != 3 {
		t.Fatalf(Format, "Count after update all", "3", fmt.Sprintf("%d", count))
	}

}

func TestSQCreate(t *testing.T) {

	params := map[string]string{
		"title":      "Test 98",
		"text":       "My text",
		"created_at": "REPLACE ME",
		"summary":    "me",
	}

	// if your model is in a package, it could be pages.Create()
	// For now to mock we just use an empty page
	id, err := (&Page{}).Create(params)
	if err != nil {
		t.Fatalf(Format, "Err on create", err)
	}

	// Now find the page and test it
	p, err := PageFind(id)
	if err != nil {
		t.Fatalf(Format, "Err on create find", err)
	}

	if p.Title != "Test 98" {
		t.Fatalf(Format, "Create page params mismatch", "Creation", p.Id)
	}

	// Check we have one left
	count, err := PageQuery().Count()

	if err != nil || count != 4 {
		t.Fatalf(Format, "Count after create", "4", fmt.Sprintf("%d", count))
	}

}

func TestSQDelete(t *testing.T) {

	p, err := PageFind(3)
	if err != nil {
		t.Fatalf(Format, "Could not find model err", "id-3", err)
	}

	err = p.Delete()

	// Check it is gone and we get an error on next find
	p, err = PageFind(3)

	if !strings.Contains(fmt.Sprintf("%s", err), "No results found") {
		t.Fatalf(Format, "Error after delete 1", "1", err)
	}

}

func TestSQDeleteAll(t *testing.T) {

	err := PageQuery().Where("id > 1").DeleteAll()
	if err != nil {
		t.Fatalf(Format, "DELETE ALL err", "delete 2 records", err)
	}

	// Check we have one left
	count, err := PageQuery().Where("id > 0").Count()

	if err != nil || count != 1 {
		t.Fatalf(Format, "Count after delete all", "1", fmt.Sprintf("%d", count))
	}

}

func TestSQTeardown(t *testing.T) {

	err := CloseDatabase()
	if err != nil {
		fmt.Println("Close DB ERROR ", err)
	}
}

// ----------------------------------
// PSQL TESTS
// ----------------------------------

func TestPQSetup(t *testing.T) {

	fmt.Println("\n---\nTESTING POSTRGRESQL\n---")

	// First execute sql (NB test might not work on windows)
	cmd := exec.Command("psql", "-dquery_test", "-f./tests/query_test_pq.sql")
	stdout, err := cmd.StdoutPipe()
	stderr, err := cmd.StderrPipe()
	err = cmd.Start()
	if err != nil {
		fmt.Println(err)
	}
	go io.Copy(os.Stdout, stdout)
	go io.Copy(os.Stderr, stderr)
	cmd.Wait()

	if err == nil {
		_ = strings.Replace("", "", "", -1)
		// Open the database
		options := map[string]string{
			"adapter": "postgres",
			"user":    "kenny",
			"db":      "query_test",
		}

		// for more detail on failure, enable debug mode on db
		options["debug"] = debug

		err = OpenDatabase(options)
		if err != nil {
			fmt.Println("db ERROR ", err)
		}

		fmt.Println("---\nQuery Testing Postgres - DB setup complete\n---")
	}

}

func TestPQFind(t *testing.T) {

	// This should work
	p, err := PageFind(1)
	if err != nil {
		t.Fatalf(Format, "Find(1)", "Model object", err)
	}

	// This should fail, so we check that
	p, err = PageFind(11)
	if err == nil {
		t.Fatalf(Format, "Find(1)", "Model object", p, err)
	}

}

func TestPQCount(t *testing.T) {

	// This should return 3
	count, err := PageQuery().Count()
	if err != nil || count != 3 {
		t.Fatalf(Format, "Count failed", "3", fmt.Sprintf("%d", count))
	}

	// This should return 2 - test limit ignored
	count, err = PageQuery().Where("id < 3").Order("id desc").Limit(100).Count()
	if err != nil || count != 2 {
		t.Fatalf(Format, "Count id < 3 failed", "2", fmt.Sprintf("%d", count))
	}

	// This should return 0
	count, err = PageQuery().Where("id > 3").Count()
	if err != nil || count != 0 {
		t.Fatalf(Format, "Count id > 3 failed", "0", fmt.Sprintf("%d", count))
	}

	// Test retrieving an array, then counting, then where
	// This should work
	q := PageQuery().Where("id > ?", 1).Order("id desc")

	count, err = q.Count()
	if err != nil || count != 2 {
		t.Fatalf(Format, "Count id > 1 failed", "2", fmt.Sprintf("%d", count), err)
	}

	// Reuse same query to get array after count
	var models []*Page
	err = q.Fetch(&models)
	if err != nil || len(models) != 2 {
		t.Fatalf(Format, "Where Array after count", "len 2", err)
	}

}

func TestPQWhere(t *testing.T) {

	var models []*Page
	q := PageQuery().Where("id > ?", 1)
	err := q.Fetch(&models)
	if err != nil || len(models) != 2 {
		t.Fatalf(Format, "Where Array", "len 2", fmt.Sprintf("%d", len(models)))
	}

}

func TestPQOrder(t *testing.T) {

	// Look for pages in reverse order
	var models []*Page
	q := PageQuery().Where("id > 1").Order("id desc")
	err := q.Fetch(&models)
	if err != nil || len(models) == 0 {
		t.Fatalf(Format, "Order test id desc", "3", fmt.Sprintf("%d", len(models)))
		return
	}

	p := models[0]
	if p.Id != 3 {
		t.Fatalf(Format, "Order test id desc", "3", fmt.Sprintf("%d", p.Id))

	}

	// Look for pages in right order
	models = make([]*Page, 0)
	q = PageQuery().Where("id < ?", 10).Where("id < ?", 100).Order("id asc")
	err = q.Fetch(&models)
	if err != nil || models == nil {
		t.Fatalf(Format, "Order test id asc", "1", err)
	}

	p = models[0]
	if p.Id != 1 {
		t.Fatalf(Format, "Order test id asc", "1", fmt.Sprintf("%d", p.Id))

	}

}

func TestPQSelect(t *testing.T) {

	var models []*Page
	q := PageQuery().Select("SELECT id,title from pages").Order("id asc")
	err := q.Fetch(&models)
	if err != nil || len(models) == 0 {
		t.Fatalf(Format, "Select error on id,title", "id,title", err)
	}
	p := models[0]
	if p.Id != 1 || p.Title != "Title 1." || len(p.Text) > 0 {
		t.Fatalf(Format, "Select id,title", "id,title only", p)
	}

}

// Some more damaging operations we execute at the end,
// to avoid having to reload the db for each test

func TestPQUpdateAll(t *testing.T) {

	err := PageQuery().UpdateAll(map[string]string{"title": "test me"})
	if err != nil {
		t.Fatalf(Format, "UPDATE ALL err", "udpate all records", err)
	}

	// Check we have all pages with same title
	count, err := PageQuery().Where("title=?", "test me").Count()

	if err != nil || count != 3 {
		t.Fatalf(Format, "Count after update all", "3", fmt.Sprintf("%d", count))
	}

}

func TestPQUpdate(t *testing.T) {

	p, err := PageFind(3)
	if err != nil {
		t.Fatalf(Format, "Update could not find model err", "id-3", err)
	}

	// Should really test updates with several strings here
	// Update each model with a different string
	// This does also check if AllowedParams is working properly to clean params
	err = p.Update(map[string]string{"title": "UPDATE 1"})

	// Check it is modified
	p, err = PageFind(3)

	if err != nil {
		t.Fatalf(Format, "Error after update 1", "updated", err)
	}

	if p.Title != "UPDATE 1" {
		t.Fatalf(Format, "Error after update 1 - Not updated properly", "UPDATE 1", p.Title)
	}

}

func TestPQCreate(t *testing.T) {

	params := map[string]string{
		//	"id":		"",
		"title":      "Test 98",
		"text":       "My text",
		"created_at": "REPLACE ME",
		"summary":    "This is my summary",
	}

	// if your model is in a package, it could be pages.Create()
	// For now to mock we just use an empty page
	id, err := (&Page{}).Create(params)
	if err != nil {
		t.Fatalf(Format, "Err on create", err)
	}

	// Now find the page and test it
	p, err := PageFind(id)
	if err != nil {
		t.Fatalf(Format, "Err on create find", err)
	}

	if p.Title != "Test 98" {
		t.Fatalf(Format, "Create page params mismatch", "Creation", p.Title)
	}

	// Check we have one left
	count, err := PageQuery().Count()

	if err != nil || count != 4 {
		t.Fatalf(Format, "Count after create", "4", fmt.Sprintf("%d", count))
	}

}

func TestPQDelete(t *testing.T) {

	p, err := PageFind(3)
	if err != nil {
		t.Fatalf(Format, "Could not find model err", "id-3", err)
	}

	err = p.Delete()

	// Check it is gone and we get an error on next find
	p, err = PageFind(3)

	if !strings.Contains(fmt.Sprintf("%s", err), "No results found") {
		t.Fatalf(Format, "Error after delete 1", "1", err)
	}

}

func TestPQDeleteAll(t *testing.T) {

	err := PageQuery().Where("id > 1").DeleteAll()
	if err != nil {
		t.Fatalf(Format, "DELETE ALL err", "delete al above 1 records", err)
	}

	// Check we have one left
	count, err := PageQuery().Count()

	if err != nil || count != 1 {
		t.Fatalf(Format, "Count after delete all above 1", "1", fmt.Sprintf("%d", count))
	}

}

/*
func TestPQSpeed(t *testing.T) {

	fmt.Println("\n---\nSpeed testing PSQL\n---")

    for i:=0;i<100000;i++ {
        // ok  	github.com/fragmenta/query	20.238s

        var models []*Page
        q := PageQuery().Select("SELECT id,title from pages").Where("id < i").Order("id asc")
    	err := q.Fetch(&models)
        if err != nil && models != nil {

        }

        // ok  	github.com/fragmenta/query	21.680s
        q := PageQuery().Select("SELECT id,title from pages").Where("id < i").Order("id asc")
    	r,err := q.Results()
        if err != nil && r != nil {

        }

    }

	fmt.Println("\n---\nSpeed testing PSQL END\n---")

}
*/

func TestPQTeardown(t *testing.T) {

	err := CloseDatabase()
	if err != nil {
		fmt.Println("Close DB ERROR ", err)
	}
}

// ----------------------------------
// MYSQL TESTS
// ----------------------------------

func TestMysqlSetup(t *testing.T) {

	fmt.Println("\n---\nTESTING Mysql\n---")

	// First execute sql

	// read whole the file
	bytes, err := ioutil.ReadFile("./tests/query_test_mysql.sql")
	if err != nil {
		panic(err)
	}
	s := string(bytes)

	cmd := exec.Command("mysql", "-u", "root", "--init", s, "query_test")
	stdout, err := cmd.StdoutPipe()
	stderr, err := cmd.StderrPipe()
	err = cmd.Start()
	if err != nil {
		fmt.Println(err)
	}
	go io.Copy(os.Stdout, stdout)
	go io.Copy(os.Stderr, stderr)
	cmd.Wait()

	if err == nil {
		_ = strings.Replace("", "", "", -1)

		// Open the database
		options := map[string]string{
			"adapter": "mysql",
			"db":      "query_test",
		}

		// for more detail on queries, enable debug mode on db
		options["debug"] = debug

		err = OpenDatabase(options)
		if err != nil {
			fmt.Println("db ERROR ", err)
		}

		fmt.Println("---\nQuery Testing Mysql - DB setup complete\n---")
	}

}

func TestMysqlFind(t *testing.T) {

	// This should work
	p, err := PageFind(1)
	if err != nil {
		t.Fatalf(Format, "Find(1)", "Model object", p)
	}

	// This should fail, so we check that
	p, err = PageFind(11)
	if err == nil {
		t.Fatalf(Format, "Find(1)", "Model object", p)
	}

}

func TestMysqlCount(t *testing.T) {

	// This should return 3
	count, err := PageQuery().Count()
	if err != nil || count != 3 {
		t.Fatalf(Format, "Count failed", "3", fmt.Sprintf("%d", count))
	}

	// This should return 2 - test limit ignored
	count, err = PageQuery().Where("id < 3").Order("id desc").Limit(100).Count()
	if err != nil || count != 2 {
		t.Fatalf(Format, "Count id < 3 failed", "2", fmt.Sprintf("%d", count))
	}

	// This should return 0
	count, err = PageQuery().Where("id > 3").Count()
	if err != nil || count != 0 {
		t.Fatalf(Format, "Count id > 3 failed", "0", fmt.Sprintf("%d", count))
	}

	// Test retrieving an array, then counting, then where
	// This should work
	q := PageQuery().Where("id > ?", 1).Order("id desc")

	count, err = q.Count()
	if err != nil || count != 2 {
		t.Fatalf(Format, "Count id > 1 failed", "2", fmt.Sprintf("%d", count), err)
	}

	// Reuse same query to get array after count
	var models []*Page
	err = q.Fetch(&models)
	if err != nil || len(models) != 2 {
		t.Fatalf(Format, "Where Array after count", "len 2", err)
	}

}

func TestMysqlWhere(t *testing.T) {

	var models []*Page
	q := PageQuery().Where("id > ?", 1)
	err := q.Fetch(&models)
	if err != nil || len(models) != 2 {
		t.Fatalf(Format, "Where Array", "len 2", fmt.Sprintf("%d", len(models)))
	}

}

func TestMysqlOrder(t *testing.T) {

	// Look for pages in reverse order
	var models []*Page
	q := PageQuery().Where("id > 1").Order("id desc")
	err := q.Fetch(&models)
	if err != nil || len(models) == 0 {
		t.Fatalf(Format, "Order test id desc", "3", fmt.Sprintf("%d", len(models)))
		return
	}

	p := models[0]
	if p.Id != 3 {
		t.Fatalf(Format, "Order test id desc", "3", fmt.Sprintf("%d", p.Id))

	}

	// Look for pages in right order
	models = make([]*Page, 0)
	q = PageQuery().Where("id < ?", 10).Where("id < ?", 100).Order("id asc")
	err = q.Fetch(&models)
	if err != nil || models == nil {
		t.Fatalf(Format, "Order test id asc", "1", err)
	}

	p = models[0]
	if p.Id != 1 {
		t.Fatalf(Format, "Order test id asc", "1", fmt.Sprintf("%d", p.Id))

	}

}

func TestMysqlSelect(t *testing.T) {

	var models []*Page
	q := PageQuery().Select("SELECT id,title from pages").Order("id asc")
	err := q.Fetch(&models)
	if err != nil || len(models) == 0 {
		t.Fatalf(Format, "Select error on id,title", "id,title", err)
	}
	p := models[0]
	if p.Id != 1 || p.Title != "Title 1." || len(p.Text) > 0 {
		t.Fatalf(Format, "Select id,title", "id,title only", p)
	}

}

func TestMysqlUpdate(t *testing.T) {

	p, err := PageFind(3)
	if err != nil {
		t.Fatalf(Format, "Update could not find model err", "id-3", err)
	}

	// Should really test updates with several strings here
	// Update each model with a different string
	// This does also check if AllowedParams is working properly to clean params
	err = p.Update(map[string]string{"title": "UPDATE 1"})

	// Check it is modified
	p, err = PageFind(3)

	if err != nil {
		t.Fatalf(Format, "Error after update 1", "updated", err)
	}

	if p.Title != "UPDATE 1" {
		t.Fatalf(Format, "Error after update 1 - Not updated properly", "UPDATE 1", p.Title)
	}

}

// Some more damaging operations we execute at the end,
// to avoid having to reload the db for each test

func TestMysqlUpdateAll(t *testing.T) {

	err := PageQuery().UpdateAll(map[string]string{"title": "test me"})
	if err != nil {
		t.Fatalf(Format, "UPDATE ALL err", "udpate all records", err)
	}

	// Check we have all pages with same title
	count, err := PageQuery().Where("title=?", "test me").Count()

	if err != nil || count != 3 {
		t.Fatalf(Format, "Count after update all", "3", fmt.Sprintf("%d", count))
	}

}

func TestMysqlCreate(t *testing.T) {

	params := map[string]string{
		"title":      "Test 98",
		"text":       "My text",
		"created_at": "REPLACE ME",
		"summary":    "me",
	}

	// if your model is in a package, it could be pages.Create()
	// For now to mock we just use an empty page
	id, err := (&Page{}).Create(params)
	if err != nil {
		t.Fatalf(Format, "Err on create", err)
	}

	// Now find the page and test it
	p, err := PageFind(id)
	if err != nil {
		t.Fatalf(Format, "Err on create find", err)
	}

	if p.Text != "My text" {
		t.Fatalf(Format, "Create page params mismatch", "Creation", p.Id)
	}

	// Check we have one left
	count, err := PageQuery().Count()

	if err != nil || count != 4 {
		t.Fatalf(Format, "Count after create", "4", fmt.Sprintf("%d", count))
	}

}

func TestMysqlDelete(t *testing.T) {

	p, err := PageFind(3)
	if err != nil {
		t.Fatalf(Format, "Could not find model err", "id-3", err)
	}
	err = p.Delete()

	// Check it is gone and we get an error on next find
	p, err = PageFind(3)
	if !strings.Contains(fmt.Sprintf("%s", err), "No results found") {
		t.Fatalf(Format, "Error after delete 1", "1", err)
	}

}

func TestMysqlDeleteAll(t *testing.T) {

	err := PageQuery().Where("id > 1").DeleteAll()
	if err != nil {
		t.Fatalf(Format, "DELETE ALL err", "delete 2 records", err)
	}

	// Check we have one left
	count, err := PageQuery().Where("id > 0").Count()

	if err != nil || count != 1 {
		t.Fatalf(Format, "Count after delete all", "1", fmt.Sprintf("%d", count))
	}

}

func TestMysqlTeardown(t *testing.T) {

	err := CloseDatabase()
	if err != nil {
		fmt.Println("Close DB ERROR ", err)
	}
}
