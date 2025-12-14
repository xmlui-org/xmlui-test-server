package cfgldr

import (
	"fmt"
	"iter"
	"strings"
)

type OrderedMap[K comparable, V any] struct {
	store map[K]V
	keys  []K
}

func NewOrderedMap[K comparable, V any]() *OrderedMap[K, V] {
	return &OrderedMap[K, V]{
		store: map[K]V{},
		keys:  []K{},
	}
}

// Clear resets an ordered map to its initial state with internal data structures
// initialized but with no keys or values.
func (o *OrderedMap[K, V]) Clear() {
	o.store = map[K]V{}
	o.keys = []K{}
}

// Get will return the value associated with the key.
// If the key doesn't exist, the second return value will be false.
func (o *OrderedMap[K, V]) Get(key K) (V, bool) {
	val, exists := o.store[key]
	return val, exists
}

// Set will store a key-value pair. If the key already exists,
// it will overwrite the existing key-value pair.
func (o *OrderedMap[K, V]) Set(key K, val V) {
	_, exists := o.store[key]
	if !exists {
		o.keys = append(o.keys, key)
	}
	o.store[key] = val
}

// Delete will remove the key and its associated value.
func (o *OrderedMap[K, V]) Delete(key K) {
	delete(o.store, key)

	// Find key in slice
	idx := -1

	for i, val := range o.keys {
		if val != key {
			continue
		}
		idx = i
		break
	}
	if idx != -1 {
		o.keys = append(o.keys[:idx], o.keys[idx+1:]...)
	}
}

// Iterator returns an iterator compatible with Go's range-over-function mechanism.
//
// Usage:
//
//	for k, v := range om.Iterator() {
//	    ...
//	}
//
// It snapshots the key order at call time so the iteration is stable even if the
// map is mutated during the loop (no locking is provided).
func (o *OrderedMap[K, V]) Iterator() iter.Seq2[K, V] {
	// Snapshot keys to avoid surprises if o.keys changes during iteration
	keys := append([]K(nil), o.keys...)

	return func(yield func(K, V) bool) {
		for _, k := range keys {
			v := o.store[k]
			if !yield(k, v) {
				return
			}
		}
	}
}

// Keys yields keys in insertion order.
//
// Usage:
//
//	for k := range om.Keys() {
//		...
//	}
func (o *OrderedMap[K, V]) Keys() iter.Seq[K] {
	keys := append([]K(nil), o.keys...)
	return func(yield func(K) bool) {
		for _, k := range keys {
			if !yield(k) {
				return
			}
		}
	}
}

// Values yields values in insertion order (aligned with Keys/Iterator).
//
// Usage:
//
//	for v := range om.GetValues() {
//		...
//	}
func (o *OrderedMap[K, V]) Values() iter.Seq[V] {
	keys := append([]K(nil), o.keys...)
	return func(yield func(V) bool) {
		for _, k := range keys {
			v := o.store[k]
			if !yield(v) {
				return
			}
		}
	}
}

// GetKeys returns a snapshot slice of the keys in insertion order.
func (o *OrderedMap[K, V]) GetKeys() []K {
	// Make a copy so caller can’t accidentally mutate our internal slice.
	keys := make([]K, len(o.keys))
	copy(keys, o.keys)
	return keys
}

// GetValues returns a slice of values in insertion order, aligned with GetKeys.
func (o *OrderedMap[K, V]) GetValues() []V {
	values := make([]V, 0, len(o.keys))
	for _, k := range o.keys {
		values = append(values, o.store[k])
	}
	return values
}

// Len returns the number of key-value pairs in the map.
func (o *OrderedMap[K, V]) Len() int {
	return len(o.keys)
}

// String converts OrderedMap to a single string.
// TODO Verify this is what we actually need for this use-case.
func (o *OrderedMap[K, V]) String() (s string) {
	sb := strings.Builder{}
	for _, k := range o.keys {
		sb.WriteString(fmt.Sprintf("%v", k))
		sb.WriteByte('=')
		sb.WriteString(fmt.Sprintf("%v", o.store[k]))
		sb.WriteByte(' ')
	}
	s = sb.String()
	if s != "" {
		s = s[:len(s)-1]
	}
	return s
}
