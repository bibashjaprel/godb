// Package index implements simple in-memory indexes used to speed up
// equality lookups against table storage. It knows nothing about the
// storage package's types — it just maps arbitrary comparable values
// to sets of row IDs — so storage can depend on index without a cycle.
package index

// HashIndex maps a column's values to the set of row IDs holding them.
// It supports O(1) average-case equality lookups at the cost of O(1)
// maintenance on every insert/update/delete that touches the indexed
// column.
type HashIndex struct {
	Column  string
	buckets map[interface{}]map[int]struct{}
}

// NewHashIndex creates an empty index over the given column.
func NewHashIndex(column string) *HashIndex {
	return &HashIndex{
		Column:  column,
		buckets: make(map[interface{}]map[int]struct{}),
	}
}

// Add records that row id has the given value in the indexed column.
func (h *HashIndex) Add(value interface{}, id int) {
	bucket, ok := h.buckets[value]
	if !ok {
		bucket = make(map[int]struct{})
		h.buckets[value] = bucket
	}
	bucket[id] = struct{}{}
}

// Remove forgets that row id has the given value.
func (h *HashIndex) Remove(value interface{}, id int) {
	bucket, ok := h.buckets[value]
	if !ok {
		return
	}
	delete(bucket, id)
	if len(bucket) == 0 {
		delete(h.buckets, value)
	}
}

// Lookup returns every row ID currently associated with value, in
// unspecified order. The returned slice is a fresh copy safe to mutate.
func (h *HashIndex) Lookup(value interface{}) []int {
	bucket, ok := h.buckets[value]
	if !ok {
		return nil
	}
	ids := make([]int, 0, len(bucket))
	for id := range bucket {
		ids = append(ids, id)
	}
	return ids
}

// Len reports how many distinct values are indexed.
func (h *HashIndex) Len() int {
	return len(h.buckets)
}
