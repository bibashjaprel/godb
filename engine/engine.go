// Package engine wires the parser's AST to the storage layer: it is
// the query executor. Execute is the single entry point a REPL or
// test needs.
package engine

import (
	"fmt"

	"godb/parser"
	"godb/storage"
)

// Engine holds the one Database this session operates on.
type Engine struct {
	DB *storage.Database
}

func New() *Engine {
	return &Engine{DB: storage.NewDatabase()}
}

// Execute parses and runs a single statement.
func (e *Engine) Execute(sql string) (*Result, error) {
	stmt, err := parser.Parse(sql)
	if err != nil {
		return nil, fmt.Errorf("parse error: %w", err)
	}

	switch s := stmt.(type) {
	case parser.CreateTableStmt:
		return e.execCreateTable(s)
	case parser.CreateIndexStmt:
		return e.execCreateIndex(s)
	case parser.InsertStmt:
		return e.execInsert(s)
	case parser.SelectStmt:
		return e.execSelect(s)
	case parser.UpdateStmt:
		return e.execUpdate(s)
	case parser.DeleteStmt:
		return e.execDelete(s)
	default:
		return nil, fmt.Errorf("unsupported statement type %T", stmt)
	}
}

func (e *Engine) execCreateTable(s parser.CreateTableStmt) (*Result, error) {
	var cols []storage.Column
	for _, c := range s.Columns {
		ct, err := storage.ParseColumnType(c.Type)
		if err != nil {
			return nil, err
		}
		cols = append(cols, storage.Column{Name: c.Name, Type: ct, PrimaryKey: c.PrimaryKey})
	}
	if _, err := e.DB.CreateTable(s.Table, storage.Schema{Columns: cols}); err != nil {
		return nil, err
	}
	return &Result{Message: fmt.Sprintf("table %q created", s.Table)}, nil
}

func (e *Engine) execCreateIndex(s parser.CreateIndexStmt) (*Result, error) {
	t, err := e.DB.GetTable(s.Table)
	if err != nil {
		return nil, err
	}
	if err := t.CreateIndex(s.Column); err != nil {
		return nil, err
	}
	return &Result{Message: fmt.Sprintf("index created on %s(%s)", s.Table, s.Column)}, nil
}

func (e *Engine) execInsert(s parser.InsertStmt) (*Result, error) {
	t, err := e.DB.GetTable(s.Table)
	if err != nil {
		return nil, err
	}

	columns := s.Columns
	if len(columns) == 0 {
		columns = t.Schema.ColumnNames()
	}

	count := 0
	for _, vals := range s.Rows {
		if len(vals) != len(columns) {
			return nil, fmt.Errorf("column count (%d) doesn't match value count (%d)", len(columns), len(vals))
		}
		row := make(storage.Row, len(columns))
		for i, col := range columns {
			row[col] = vals[i].Value
		}
		if _, err := t.Insert(row); err != nil {
			return nil, err
		}
		count++
	}
	return &Result{Message: fmt.Sprintf("%d row(s) inserted", count)}, nil
}

func (e *Engine) execSelect(s parser.SelectStmt) (*Result, error) {
	t, err := e.DB.GetTable(s.Table)
	if err != nil {
		return nil, err
	}

	records, err := matchingRecords(t, s.Where)
	if err != nil {
		return nil, err
	}

	columns := s.Columns
	if len(columns) == 1 && columns[0] == "*" {
		columns = t.Schema.ColumnNames()
	} else {
		for _, c := range columns {
			if !t.Schema.HasColumn(c) {
				return nil, fmt.Errorf("no such column %q", c)
			}
		}
	}

	rows := make([][]interface{}, 0, len(records))
	for _, rec := range records {
		vals := make([]interface{}, len(columns))
		for i, c := range columns {
			vals[i] = rec.Row[c]
		}
		rows = append(rows, vals)
	}
	return &Result{Columns: columns, Rows: rows}, nil
}

func (e *Engine) execUpdate(s parser.UpdateStmt) (*Result, error) {
	t, err := e.DB.GetTable(s.Table)
	if err != nil {
		return nil, err
	}
	records, err := matchingRecords(t, s.Where)
	if err != nil {
		return nil, err
	}
	updates := make(map[string]interface{}, len(s.Set))
	for col, lit := range s.Set {
		updates[col] = lit.Value
	}
	for _, rec := range records {
		if err := t.Update(rec.ID, updates); err != nil {
			return nil, err
		}
	}
	return &Result{Message: fmt.Sprintf("%d row(s) updated", len(records))}, nil
}

func (e *Engine) execDelete(s parser.DeleteStmt) (*Result, error) {
	t, err := e.DB.GetTable(s.Table)
	if err != nil {
		return nil, err
	}
	records, err := matchingRecords(t, s.Where)
	if err != nil {
		return nil, err
	}
	for _, rec := range records {
		if err := t.Delete(rec.ID); err != nil {
			return nil, err
		}
	}
	return &Result{Message: fmt.Sprintf("%d row(s) deleted", len(records))}, nil
}
