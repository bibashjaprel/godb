package storage

import (
	"fmt"
	"sort"
	"sync"

	"godb/index"
)

// Record pairs a row with the internal ID it was stored under. IDs are
// never reused, even after a delete, so an index entry always points
// at either a live row or nothing.
type Record struct {
	ID  int
	Row Row
}

// Table is an in-memory, schema-enforcing collection of rows plus
// whatever hash indexes have been built on top of it.
type Table struct {
	Name   string
	Schema Schema

	mu      sync.RWMutex
	rows    map[int]Row
	order   []int // insertion order of live IDs
	nextID  int
	indexes map[string]*index.HashIndex
}

// NewTable creates an empty table with the given schema. If the schema
// marks a column as PRIMARY KEY, a hash index is created on it
// automatically.
func NewTable(name string, schema Schema) *Table {
	t := &Table{
		Name:    name,
		Schema:  schema,
		rows:    make(map[int]Row),
		indexes: make(map[string]*index.HashIndex),
	}
	for _, c := range schema.Columns {
		if c.PrimaryKey {
			t.indexes[c.Name] = index.NewHashIndex(c.Name)
		}
	}
	return t
}

// CreateIndex builds a hash index over an existing column, backfilling
// it from every row currently in the table. It is a no-op (but still an
// error) if the column doesn't exist, and idempotent if the index
// already exists.
func (t *Table) CreateIndex(column string) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if !t.Schema.HasColumn(column) {
		return fmt.Errorf("table %q has no column %q", t.Name, column)
	}
	if _, ok := t.indexes[column]; ok {
		return nil
	}
	idx := index.NewHashIndex(column)
	for id, row := range t.rows {
		idx.Add(row[column], id)
	}
	t.indexes[column] = idx
	return nil
}

// IndexFor returns the index on column, if one exists.
func (t *Table) IndexFor(column string) (*index.HashIndex, bool) {
	t.mu.RLock()
	defer t.mu.RUnlock()
	idx, ok := t.indexes[column]
	return idx, ok
}

// Insert validates row against the schema, assigns it a fresh ID,
// stores it, and updates any affected indexes.
func (t *Table) Insert(row Row) (int, error) {
	t.mu.Lock()
	defer t.mu.Unlock()

	checked := make(Row, len(t.Schema.Columns))
	for _, col := range t.Schema.Columns {
		v, present := row[col.Name]
		if !present {
			return 0, fmt.Errorf("missing value for column %q", col.Name)
		}
		cv, err := t.Schema.CheckValue(col.Name, v)
		if err != nil {
			return 0, err
		}
		checked[col.Name] = cv
	}
	for k := range row {
		if !t.Schema.HasColumn(k) {
			return 0, fmt.Errorf("no such column %q on table %q", k, t.Name)
		}
	}

	id := t.nextID
	t.nextID++
	t.rows[id] = checked
	t.order = append(t.order, id)

	for col, idx := range t.indexes {
		idx.Add(checked[col], id)
	}
	return id, nil
}

// Get fetches a single row by internal ID.
func (t *Table) Get(id int) (Row, bool) {
	t.mu.RLock()
	defer t.mu.RUnlock()
	row, ok := t.rows[id]
	if !ok {
		return nil, false
	}
	return row.Clone(), true
}

// Update applies a partial set of column assignments to the row with
// the given ID, keeping indexes in sync.
func (t *Table) Update(id int, updates map[string]interface{}) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	row, ok := t.rows[id]
	if !ok {
		return fmt.Errorf("no row with id %d", id)
	}

	checkedUpdates := make(map[string]interface{}, len(updates))
	for col, v := range updates {
		cv, err := t.Schema.CheckValue(col, v)
		if err != nil {
			return err
		}
		checkedUpdates[col] = cv
	}

	// Remove stale index entries, apply the update, then re-add.
	for col, idx := range t.indexes {
		if _, touched := checkedUpdates[col]; touched {
			idx.Remove(row[col], id)
		}
	}
	for col, v := range checkedUpdates {
		row[col] = v
	}
	t.rows[id] = row
	for col, idx := range t.indexes {
		if _, touched := checkedUpdates[col]; touched {
			idx.Add(row[col], id)
		}
	}
	return nil
}

// Delete removes a row by ID, cleaning up any index entries.
func (t *Table) Delete(id int) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	row, ok := t.rows[id]
	if !ok {
		return fmt.Errorf("no row with id %d", id)
	}
	for col, idx := range t.indexes {
		idx.Remove(row[col], id)
	}
	delete(t.rows, id)

	// Drop id from the order slice.
	for i, oid := range t.order {
		if oid == id {
			t.order = append(t.order[:i], t.order[i+1:]...)
			break
		}
	}
	return nil
}

// Scan returns every live row in insertion order.
func (t *Table) Scan() []Record {
	t.mu.RLock()
	defer t.mu.RUnlock()
	out := make([]Record, 0, len(t.order))
	for _, id := range t.order {
		out = append(out, Record{ID: id, Row: t.rows[id].Clone()})
	}
	return out
}

// ScanIDs returns the given row IDs as Records, skipping any that no
// longer exist. Used by the query engine after an index lookup.
func (t *Table) ScanIDs(ids []int) []Record {
	t.mu.RLock()
	defer t.mu.RUnlock()
	sorted := append([]int(nil), ids...)
	sort.Ints(sorted)
	out := make([]Record, 0, len(sorted))
	for _, id := range sorted {
		if row, ok := t.rows[id]; ok {
			out = append(out, Record{ID: id, Row: row.Clone()})
		}
	}
	return out
}

// Len reports how many live rows the table holds.
func (t *Table) Len() int {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return len(t.rows)
}
