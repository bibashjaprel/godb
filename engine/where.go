package engine

import (
	"fmt"

	"godb/parser"
	"godb/storage"
)

// matchingRecords finds every row in t that satisfies every condition
// in where (conditions are AND-ed together). When the first condition
// is an equality test on an indexed column, it uses the index to
// narrow the candidate set before falling back to a linear scan for
// any remaining conditions — the one place this "core" engine gets to
// show off why indexes exist.
func matchingRecords(t *storage.Table, where []parser.Condition) ([]storage.Record, error) {
	if err := validateConditions(t, where); err != nil {
		return nil, err
	}

	var candidates []storage.Record
	remaining := where

	if len(where) > 0 && where[0].Op == parser.EQ {
		if idx, ok := t.IndexFor(where[0].Column); ok {
			ids := idx.Lookup(where[0].Value.Value)
			candidates = t.ScanIDs(ids)
			remaining = where[1:]
		}
	}
	if candidates == nil {
		candidates = t.Scan()
	}

	if len(remaining) == 0 {
		return candidates, nil
	}

	out := make([]storage.Record, 0, len(candidates))
	for _, rec := range candidates {
		ok, err := matchesAll(rec.Row, remaining)
		if err != nil {
			return nil, err
		}
		if ok {
			out = append(out, rec)
		}
	}
	return out, nil
}

func validateConditions(t *storage.Table, where []parser.Condition) error {
	for _, c := range where {
		if !t.Schema.HasColumn(c.Column) {
			return fmt.Errorf("no such column %q", c.Column)
		}
	}
	return nil
}

func matchesAll(row storage.Row, conds []parser.Condition) (bool, error) {
	for _, c := range conds {
		ok, err := matches(row[c.Column], c.Op, c.Value.Value)
		if err != nil {
			return false, err
		}
		if !ok {
			return false, nil
		}
	}
	return true, nil
}

func matches(actual interface{}, op parser.TokenType, target interface{}) (bool, error) {
	switch a := actual.(type) {
	case int64:
		b, ok := target.(int64)
		if !ok {
			return false, fmt.Errorf("cannot compare INT column to %T", target)
		}
		switch op {
		case parser.EQ:
			return a == b, nil
		case parser.NEQ:
			return a != b, nil
		case parser.LT:
			return a < b, nil
		case parser.LE:
			return a <= b, nil
		case parser.GT:
			return a > b, nil
		case parser.GE:
			return a >= b, nil
		}
	case string:
		b, ok := target.(string)
		if !ok {
			return false, fmt.Errorf("cannot compare TEXT column to %T", target)
		}
		switch op {
		case parser.EQ:
			return a == b, nil
		case parser.NEQ:
			return a != b, nil
		case parser.LT:
			return a < b, nil
		case parser.LE:
			return a <= b, nil
		case parser.GT:
			return a > b, nil
		case parser.GE:
			return a >= b, nil
		}
	case bool:
		b, ok := target.(bool)
		if !ok {
			return false, fmt.Errorf("cannot compare BOOL column to %T", target)
		}
		switch op {
		case parser.EQ:
			return a == b, nil
		case parser.NEQ:
			return a != b, nil
		default:
			return false, fmt.Errorf("BOOL columns only support = and !=")
		}
	}
	return false, fmt.Errorf("unsupported value type %T in WHERE clause", actual)
}
