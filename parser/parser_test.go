package parser

import "testing"

func TestParseCreateTable(t *testing.T) {
	stmt, err := Parse(`CREATE TABLE users (id INT PRIMARY KEY, name TEXT, active BOOL);`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	ct, ok := stmt.(CreateTableStmt)
	if !ok {
		t.Fatalf("expected CreateTableStmt, got %T", stmt)
	}
	if ct.Table != "users" || len(ct.Columns) != 3 {
		t.Fatalf("unexpected result: %+v", ct)
	}
	if !ct.Columns[0].PrimaryKey {
		t.Errorf("expected id to be marked PRIMARY KEY")
	}
}

func TestParseInsertMultiRow(t *testing.T) {
	stmt, err := Parse(`INSERT INTO users (id, name) VALUES (1, 'ada'), (2, 'grace');`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	ins, ok := stmt.(InsertStmt)
	if !ok {
		t.Fatalf("expected InsertStmt, got %T", stmt)
	}
	if len(ins.Rows) != 2 {
		t.Fatalf("expected 2 rows, got %d", len(ins.Rows))
	}
	if ins.Rows[1][1].Value != "grace" {
		t.Errorf("expected second row's name to be 'grace', got %v", ins.Rows[1][1].Value)
	}
}

func TestParseSelectWithWhere(t *testing.T) {
	stmt, err := Parse(`SELECT id, name FROM users WHERE active = TRUE AND id > 1;`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	sel, ok := stmt.(SelectStmt)
	if !ok {
		t.Fatalf("expected SelectStmt, got %T", stmt)
	}
	if len(sel.Where) != 2 {
		t.Fatalf("expected 2 conditions, got %d", len(sel.Where))
	}
	if sel.Where[0].Op != EQ || sel.Where[1].Op != GT {
		t.Errorf("unexpected operators: %+v", sel.Where)
	}
}

func TestParseUpdate(t *testing.T) {
	stmt, err := Parse(`UPDATE users SET active = FALSE WHERE id = 1;`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	upd, ok := stmt.(UpdateStmt)
	if !ok {
		t.Fatalf("expected UpdateStmt, got %T", stmt)
	}
	if upd.Set["active"].Value != false {
		t.Errorf("expected active=false, got %v", upd.Set["active"].Value)
	}
}

func TestParseDelete(t *testing.T) {
	stmt, err := Parse(`DELETE FROM users WHERE id != 1;`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	del, ok := stmt.(DeleteStmt)
	if !ok {
		t.Fatalf("expected DeleteStmt, got %T", stmt)
	}
	if del.Where[0].Op != NEQ {
		t.Errorf("expected NEQ operator, got %v", del.Where[0].Op)
	}
}

func TestParseErrorOnGarbage(t *testing.T) {
	if _, err := Parse(`SELECT FROM WHERE;`); err == nil {
		t.Fatalf("expected a parse error for malformed input")
	}
}
