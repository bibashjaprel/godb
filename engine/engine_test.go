package engine

import "testing"

func mustExec(t *testing.T, e *Engine, sql string) *Result {
	t.Helper()
	res, err := e.Execute(sql)
	if err != nil {
		t.Fatalf("exec %q failed: %v", sql, err)
	}
	return res
}

func setupUsers(t *testing.T) *Engine {
	e := New()
	mustExec(t, e, `CREATE TABLE users (id INT PRIMARY KEY, name TEXT, age INT);`)
	mustExec(t, e, `INSERT INTO users (id, name, age) VALUES (1, 'ada', 30), (2, 'grace', 40), (3, 'linus', 25);`)
	return e
}

func TestEndToEndCRUD(t *testing.T) {
	e := setupUsers(t)

	res := mustExec(t, e, `SELECT id, name FROM users WHERE age > 28;`)
	if len(res.Rows) != 2 {
		t.Fatalf("expected 2 rows for age > 28, got %d: %v", len(res.Rows), res.Rows)
	}

	mustExec(t, e, `UPDATE users SET age = 31 WHERE name = 'ada';`)
	res = mustExec(t, e, `SELECT age FROM users WHERE id = 1;`)
	if res.Rows[0][0] != int64(31) {
		t.Fatalf("expected age 31 after update, got %v", res.Rows[0][0])
	}

	mustExec(t, e, `DELETE FROM users WHERE id = 3;`)
	res = mustExec(t, e, `SELECT * FROM users;`)
	if len(res.Rows) != 2 {
		t.Fatalf("expected 2 rows after delete, got %d", len(res.Rows))
	}
}

func TestSelectStar(t *testing.T) {
	e := setupUsers(t)
	res := mustExec(t, e, `SELECT * FROM users WHERE id = 2;`)
	if len(res.Columns) != 3 {
		t.Fatalf("expected 3 columns from *, got %v", res.Columns)
	}
	if len(res.Rows) != 1 || res.Rows[0][1] != "grace" {
		t.Fatalf("unexpected row: %v", res.Rows)
	}
}

func TestPrimaryKeyIndexIsUsedForEquality(t *testing.T) {
	e := setupUsers(t)
	tbl, err := e.DB.GetTable("users")
	if err != nil {
		t.Fatalf("get table: %v", err)
	}
	if _, ok := tbl.IndexFor("id"); !ok {
		t.Fatalf("expected an automatic index on primary key id")
	}
	res := mustExec(t, e, `SELECT name FROM users WHERE id = 2;`)
	if len(res.Rows) != 1 || res.Rows[0][0] != "grace" {
		t.Fatalf("expected grace via indexed lookup, got %v", res.Rows)
	}
}

func TestExplicitIndexOnNonKeyColumn(t *testing.T) {
	e := setupUsers(t)
	mustExec(t, e, `CREATE INDEX ON users (name);`)
	res := mustExec(t, e, `SELECT id FROM users WHERE name = 'linus';`)
	if len(res.Rows) != 1 || res.Rows[0][0] != int64(3) {
		t.Fatalf("expected id 3 for linus, got %v", res.Rows)
	}
}

func TestErrorsPropagate(t *testing.T) {
	e := setupUsers(t)
	if _, err := e.Execute(`SELECT nope FROM users;`); err == nil {
		t.Fatalf("expected error for unknown column")
	}
	if _, err := e.Execute(`SELECT * FROM ghost;`); err == nil {
		t.Fatalf("expected error for unknown table")
	}
	// Note: this engine has no uniqueness constraint on PRIMARY KEY —
	// it only auto-indexes it. A duplicate id is accepted, and both
	// rows show up under an index lookup for that value.
	if _, err := e.Execute(`INSERT INTO users (id, name, age) VALUES (1, 'dup', 1);`); err != nil {
		t.Fatalf("expected duplicate primary key insert to succeed (no uniqueness constraint), got error: %v", err)
	}
}
