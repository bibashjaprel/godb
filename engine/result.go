package engine

// Result is what Execute returns for any statement. For SELECT,
// Columns and Rows are populated. For everything else, Message
// describes what happened (e.g. "1 row inserted").
type Result struct {
	Columns []string
	Rows    [][]interface{}
	Message string
}
