package parser

// Statement is implemented by every top-level statement type this
// parser can produce.
type Statement interface {
	statementNode()
}

// ColumnDef is one column in a CREATE TABLE statement.
type ColumnDef struct {
	Name       string
	Type       string // "INT", "TEXT", or "BOOL"
	PrimaryKey bool
}

// CreateTableStmt represents: CREATE TABLE name (col type [PRIMARY KEY], ...);
type CreateTableStmt struct {
	Table   string
	Columns []ColumnDef
}

// CreateIndexStmt represents: CREATE INDEX ON table (column);
type CreateIndexStmt struct {
	Table  string
	Column string
}

// InsertStmt represents: INSERT INTO table [(col, ...)] VALUES (v, ...), (v, ...);
type InsertStmt struct {
	Table   string
	Columns []string // empty means "use the table's schema order"
	Rows    [][]Literal
}

// SelectStmt represents: SELECT col, ... | * FROM table [WHERE cond AND cond ...];
type SelectStmt struct {
	Table   string
	Columns []string // ["*"] means all columns
	Where   []Condition
}

// UpdateStmt represents: UPDATE table SET col = v, ... [WHERE cond AND cond ...];
type UpdateStmt struct {
	Table string
	Set   map[string]Literal
	Where []Condition
}

// DeleteStmt represents: DELETE FROM table [WHERE cond AND cond ...];
type DeleteStmt struct {
	Table string
	Where []Condition
}

func (CreateTableStmt) statementNode() {}
func (CreateIndexStmt) statementNode() {}
func (InsertStmt) statementNode()      {}
func (SelectStmt) statementNode()      {}
func (UpdateStmt) statementNode()      {}
func (DeleteStmt) statementNode()      {}

// Condition is a single "column OP literal" test. Multiple Conditions
// in a WHERE clause are always AND-ed together — this engine doesn't
// support OR or parentheses in WHERE clauses, by design (core scope).
type Condition struct {
	Column string
	Op     TokenType // EQ, NEQ, LT, LE, GT, GE
	Value  Literal
}

// Literal is a parsed constant value: int64, string, or bool.
type Literal struct {
	Value interface{}
}
