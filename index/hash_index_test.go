package index

import "testing"

func TestAddLookupRemove(t *testing.T) {
	idx := NewHashIndex("id")
	idx.Add(int64(1), 100)
	idx.Add(int64(1), 101) // two rows can share a value
	idx.Add(int64(2), 200)

	got := idx.Lookup(int64(1))
	if len(got) != 2 {
		t.Fatalf("expected 2 ids for value 1, got %v", got)
	}

	idx.Remove(int64(1), 100)
	got = idx.Lookup(int64(1))
	if len(got) != 1 || got[0] != 101 {
		t.Fatalf("expected [101] after removing 100, got %v", got)
	}

	idx.Remove(int64(1), 101)
	if got := idx.Lookup(int64(1)); len(got) != 0 {
		t.Fatalf("expected empty bucket to be cleaned up, got %v", got)
	}
	if idx.Len() != 1 {
		t.Fatalf("expected only value 2's bucket left, Len()=%d", idx.Len())
	}
}

func TestLookupMissingValue(t *testing.T) {
	idx := NewHashIndex("id")
	if got := idx.Lookup("nope"); got != nil {
		t.Fatalf("expected nil for missing value, got %v", got)
	}
}
