package storage

import (
	"fmt"
	"sync"
)

// Database is a named collection of tables — the top-level handle the
// query engine operates on.
type Database struct {
	mu     sync.RWMutex
	tables map[string]*Table
}

func NewDatabase() *Database {
	return &Database{tables: make(map[string]*Table)}
}

func (d *Database) CreateTable(name string, schema Schema) (*Table, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if _, exists := d.tables[name]; exists {
		return nil, fmt.Errorf("table %q already exists", name)
	}
	t := NewTable(name, schema)
	d.tables[name] = t
	return t, nil
}

func (d *Database) GetTable(name string) (*Table, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	t, ok := d.tables[name]
	if !ok {
		return nil, fmt.Errorf("no such table %q", name)
	}
	return t, nil
}

func (d *Database) DropTable(name string) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	if _, ok := d.tables[name]; !ok {
		return fmt.Errorf("no such table %q", name)
	}
	delete(d.tables, name)
	return nil
}

// TableNames returns every table name currently defined, for
// introspection commands like a REPL's \dt.
func (d *Database) TableNames() []string {
	d.mu.RLock()
	defer d.mu.RUnlock()
	names := make([]string, 0, len(d.tables))
	for n := range d.tables {
		names = append(names, n)
	}
	return names
}
