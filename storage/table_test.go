package storage

import "testing"

func sampleSchema() Schema {
	return Schema{Columns: []Column{
		{Name: "id", Type: TypeInt, PrimaryKey: true},
		{Name: "name", Type: TypeText},
		{Name: "active", Type: TypeBool},
	}}
}

func TestInsertAndGet(t *testing.T) {
	tbl := NewTable("users", sampleSchema())
	id, err := tbl.Insert(Row{"id": int64(1), "name": "ada", "active": true})
	if err != nil {
		t.Fatalf("insert failed: %v", err)
	}
	row, ok := tbl.Get(id)
	if !ok {
		t.Fatalf("expected row to exist")
	}
	if row["name"] != "ada" {
		t.Errorf("got name %v, want ada", row["name"])
	}
}

func TestInsertWrongType(t *testing.T) {
	tbl := NewTable("users", sampleSchema())
	_, err := tbl.Insert(Row{"id": "not-an-int", "name": "ada", "active": true})
	if err == nil {
		t.Fatalf("expected a type error, got nil")
	}
}

func TestInsertMissingColumn(t *testing.T) {
	tbl := NewTable("users", sampleSchema())
	_, err := tbl.Insert(Row{"id": int64(1), "name": "ada"})
	if err == nil {
		t.Fatalf("expected an error for missing column, got nil")
	}
}

func TestUpdateAndDelete(t *testing.T) {
	tbl := NewTable("users", sampleSchema())
	id, _ := tbl.Insert(Row{"id": int64(1), "name": "ada", "active": true})

	if err := tbl.Update(id, map[string]interface{}{"active": false}); err != nil {
		t.Fatalf("update failed: %v", err)
	}
	row, _ := tbl.Get(id)
	if row["active"] != false {
		t.Errorf("got active=%v, want false", row["active"])
	}

	if err := tbl.Delete(id); err != nil {
		t.Fatalf("delete failed: %v", err)
	}
	if _, ok := tbl.Get(id); ok {
		t.Errorf("expected row to be gone after delete")
	}
	if tbl.Len() != 0 {
		t.Errorf("expected 0 rows, got %d", tbl.Len())
	}
}

func TestScanPreservesInsertionOrder(t *testing.T) {
	tbl := NewTable("users", sampleSchema())
	tbl.Insert(Row{"id": int64(1), "name": "a", "active": true})
	tbl.Insert(Row{"id": int64(2), "name": "b", "active": true})
	tbl.Insert(Row{"id": int64(3), "name": "c", "active": true})

	recs := tbl.Scan()
	if len(recs) != 3 {
		t.Fatalf("expected 3 rows, got %d", len(recs))
	}
	want := []string{"a", "b", "c"}
	for i, rec := range recs {
		if rec.Row["name"] != want[i] {
			t.Errorf("position %d: got %v, want %v", i, rec.Row["name"], want[i])
		}
	}
}

func TestPrimaryKeyIndexUpdatesOnWrite(t *testing.T) {
	tbl := NewTable("users", sampleSchema())
	id, _ := tbl.Insert(Row{"id": int64(42), "name": "ada", "active": true})

	idx, ok := tbl.IndexFor("id")
	if !ok {
		t.Fatalf("expected an automatic index on the primary key")
	}
	ids := idx.Lookup(int64(42))
	if len(ids) != 1 || ids[0] != id {
		t.Fatalf("expected lookup(42) == [%d], got %v", id, ids)
	}

	tbl.Delete(id)
	if ids := idx.Lookup(int64(42)); len(ids) != 0 {
		t.Errorf("expected index entry to be removed after delete, got %v", ids)
	}
}

func TestCreateIndexBackfills(t *testing.T) {
	tbl := NewTable("users", sampleSchema())
	tbl.Insert(Row{"id": int64(1), "name": "ada", "active": true})
	tbl.Insert(Row{"id": int64(2), "name": "grace", "active": true})

	if err := tbl.CreateIndex("name"); err != nil {
		t.Fatalf("create index failed: %v", err)
	}
	idx, ok := tbl.IndexFor("name")
	if !ok {
		t.Fatalf("expected index to exist")
	}
	if ids := idx.Lookup("grace"); len(ids) != 1 {
		t.Errorf("expected backfilled index to find 'grace', got %v", ids)
	}
}
