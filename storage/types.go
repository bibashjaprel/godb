package storage

import "fmt"

// ColumnType is the type of a column in a table's schema.
type ColumnType int

const (
	TypeInt ColumnType = iota
	TypeText
	TypeBool
)

func (t ColumnType) String() string {
	switch t {
	case TypeInt:
		return "INT"
	case TypeText:
		return "TEXT"
	case TypeBool:
		return "BOOL"
	default:
		return "UNKNOWN"
	}
}

// ParseColumnType maps a SQL type keyword (already uppercased) to a ColumnType.
func ParseColumnType(s string) (ColumnType, error) {
	switch s {
	case "INT", "INTEGER":
		return TypeInt, nil
	case "TEXT", "STRING", "VARCHAR":
		return TypeText, nil
	case "BOOL", "BOOLEAN":
		return TypeBool, nil
	default:
		return 0, fmt.Errorf("unknown column type %q", s)
	}
}

// Column describes a single column in a table schema.
type Column struct {
	Name       string
	Type       ColumnType
	PrimaryKey bool
}

// Schema is an ordered list of columns. Order matters for positional inserts.
type Schema struct {
	Columns []Column
}

func (s Schema) ColumnNames() []string {
	names := make([]string, len(s.Columns))
	for i, c := range s.Columns {
		names[i] = c.Name
	}
	return names
}

func (s Schema) HasColumn(name string) bool {
	_, ok := s.find(name)
	return ok
}

func (s Schema) find(name string) (Column, bool) {
	for _, c := range s.Columns {
		if c.Name == name {
			return c, true
		}
	}
	return Column{}, false
}

// CheckValue verifies that value is a legal value for the named column,
// and returns it coerced to the canonical Go type used internally
// (int64 for TypeInt, string for TypeText, bool for TypeBool).
func (s Schema) CheckValue(colName string, value interface{}) (interface{}, error) {
	col, ok := s.find(colName)
	if !ok {
		return nil, fmt.Errorf("no such column %q", colName)
	}
	switch col.Type {
	case TypeInt:
		switch v := value.(type) {
		case int64:
			return v, nil
		case int:
			return int64(v), nil
		default:
			return nil, fmt.Errorf("column %q expects INT, got %T", colName, value)
		}
	case TypeText:
		if v, ok := value.(string); ok {
			return v, nil
		}
		return nil, fmt.Errorf("column %q expects TEXT, got %T", colName, value)
	case TypeBool:
		if v, ok := value.(bool); ok {
			return v, nil
		}
		return nil, fmt.Errorf("column %q expects BOOL, got %T", colName, value)
	}
	return nil, fmt.Errorf("unhandled column type for %q", colName)
}

// Row is a single record, keyed by column name.
type Row map[string]interface{}

// Clone returns a shallow copy of the row (values are scalars, so this is a full copy).
func (r Row) Clone() Row {
	out := make(Row, len(r))
	for k, v := range r {
		out[k] = v
	}
	return out
}
