package query

// Common chainable finders used by fragmenta models
// These finders belong in other packages, but end up here
// because we can't reopen the query package to add them.
// They require the eponymous columns

// These are obsoleted by query.Apply() and should be defined as queryfuncs in the packages instead

/*

// Define a default order by updated_at, then created_at, then id
func (q *Query) Ordered() *Query {
	return q.Order("updated_at desc, created_at desc, id desc")
}

// Define a where query on role column - .Role(role.EDITOR)
func (q *Query) Role(r int64) *Query {
	return q.Where("role=?",r)
}

// Define a where query on url column - .Url(params["url"])
func (q *Query) Url(u string) *Query {
	return q.Where("url=?",u)
}

// Define a where query on status column - .Status()
func (q *Query) Status(s int64) *Query {
	return q.Where("status=?",s)
}

// Define a where query on status column where status >= 100
// This depends on status published being 100, but allows chaining if we put it here
func (q *Query) Published() *Query {
	return q.Where("status>=?", 100)
}


*/
