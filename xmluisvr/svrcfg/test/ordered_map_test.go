package test

import (
	"testing"

	"github.com/xmlui-org/xmlui-test-server/xmluisvr/svrcfg"
)

func TestOrderedMap_NewOrderedMap(t *testing.T) {
	om := cfgldr.NewOrderedMap[string, int]()
	if om == nil {
		t.Fatal("cfgldr.NewOrderedMap returned nil")
	}
	if om.Len() != 0 {
		t.Error("map should be empty initially")
	}
}

func TestOrderedMap_SetAndGet(t *testing.T) {
	om := cfgldr.NewOrderedMap[string, int]()

	// Test setting and getting values
	om.Set("first", 1)
	om.Set("second", 2)
	om.Set("third", 3)

	// Test Get with existing keys
	if val, exists := om.Get("first"); !exists || val != 1 {
		t.Errorf("Get(first) = %v, %v; want 1, true", val, exists)
	}
	if val, exists := om.Get("second"); !exists || val != 2 {
		t.Errorf("Get(second) = %v, %v; want 2, true", val, exists)
	}
	if val, exists := om.Get("third"); !exists || val != 3 {
		t.Errorf("Get(third) = %v, %v; want 3, true", val, exists)
	}

	// Test Get with non-existing key
	if val, exists := om.Get("nonexistent"); exists {
		t.Errorf("Get(nonexistent) = %v, %v; want 0, false", val, exists)
	}

	// Test overwriting existing key
	om.Set("first", 10)
	if val, exists := om.Get("first"); !exists || val != 10 {
		t.Errorf("Get(first) after overwrite = %v, %v; want 10, true", val, exists)
	}

	// Verify order is preserved after overwrite
	keys := om.GetKeys()
	expected := []string{"first", "second", "third"}
	if len(keys) != len(expected) {
		t.Fatalf("keys length = %d; want %d", len(keys), len(expected))
	}
	for i, key := range keys {
		if key != expected[i] {
			t.Errorf("keys[%d] = %s; want %s", i, key, expected[i])
		}
	}
}

func TestOrderedMap_Delete(t *testing.T) {
	om := cfgldr.NewOrderedMap[string, int]()

	// Set up test data
	om.Set("a", 1)
	om.Set("b", 2)
	om.Set("c", 3)
	om.Set("d", 4)

	// Delete middle element
	om.Delete("b")

	// Verify it's gone
	if _, exists := om.Get("b"); exists {
		t.Error("Delete failed: key 'b' still exists")
	}

	// Verify remaining keys and their order
	keys := om.GetKeys()
	expected := []string{"a", "c", "d"}
	if len(keys) != len(expected) {
		t.Fatalf("keys length after delete = %d; want %d", len(keys), len(expected))
	}
	for i, key := range keys {
		if key != expected[i] {
			t.Errorf("keys[%d] after delete = %s; want %s", i, key, expected[i])
		}
	}

	// Delete first element
	om.Delete("a")
	keys = om.GetKeys()
	expected = []string{"c", "d"}
	if len(keys) != len(expected) {
		t.Fatalf("keys length after second delete = %d; want %d", len(keys), len(expected))
	}
	for i, key := range keys {
		if key != expected[i] {
			t.Errorf("keys[%d] after second delete = %s; want %s", i, key, expected[i])
		}
	}

	// Delete last element
	om.Delete("d")
	keys = om.GetKeys()
	expected = []string{"c"}
	if len(keys) != len(expected) {
		t.Fatalf("keys length after third delete = %d; want %d", len(keys), len(expected))
	}
	for i, key := range keys {
		if key != expected[i] {
			t.Errorf("keys[%d] after third delete = %s; want %s", i, key, expected[i])
		}
	}

	// Delete non-existent key (should not panic)
	om.Delete("nonexistent")
}

func TestOrderedMap_Clear(t *testing.T) {
	om := cfgldr.NewOrderedMap[string, int]()

	// Set up test data
	om.Set("a", 1)
	om.Set("b", 2)
	om.Set("c", 3)

	// Clear
	om.Clear()

	// Verify empty
	if om.Len() != 0 {
		t.Errorf("map length after clear = %d; want 0", om.Len())
	}

	// Verify Get returns false for previously existing keys
	if _, exists := om.Get("a"); exists {
		t.Error("Get(a) after clear should return false")
	}
}

func TestOrderedMap_Iterator(t *testing.T) {
	om := cfgldr.NewOrderedMap[string, int]()

	// Test empty iterator
	count := 0
	for k, v := range om.Iterator() {
		count++
		t.Errorf("unexpected iteration on empty map: %s=%d", k, v)
	}
	if count != 0 {
		t.Errorf("empty iterator count = %d; want 0", count)
	}

	// Set up test data
	expected := map[string]int{
		"first":  1,
		"second": 2,
		"third":  3,
	}
	expectedOrder := []string{"first", "second", "third"}

	for _, key := range expectedOrder {
		om.Set(key, expected[key])
	}

	// Test iterator order and values
	i := 0
	for k, v := range om.Iterator() {
		if i >= len(expectedOrder) {
			t.Errorf("iterator returned more items than expected")
			break
		}
		expectedKey := expectedOrder[i]
		expectedVal := expected[expectedKey]
		if k != expectedKey || v != expectedVal {
			t.Errorf("iterator[%d] = %s=%d; want %s=%d", i, k, v, expectedKey, expectedVal)
		}
		i++
	}
	if i != len(expectedOrder) {
		t.Errorf("iterator returned %d items; want %d", i, len(expectedOrder))
	}
}

func TestOrderedMap_Keys(t *testing.T) {
	om := cfgldr.NewOrderedMap[string, int]()

	// Test empty
	count := 0
	for k := range om.Keys() {
		count++
		t.Errorf("unexpected key on empty map: %s", k)
	}
	if count != 0 {
		t.Errorf("empty Keys() count = %d; want 0", count)
	}

	// Set up test data
	expectedOrder := []string{"alpha", "beta", "gamma"}
	for i, key := range expectedOrder {
		om.Set(key, i)
	}

	// Test Keys iterator
	i := 0
	for k := range om.Keys() {
		if i >= len(expectedOrder) {
			t.Errorf("Keys() returned more items than expected")
			break
		}
		if k != expectedOrder[i] {
			t.Errorf("Keys()[%d] = %s; want %s", i, k, expectedOrder[i])
		}
		i++
	}
	if i != len(expectedOrder) {
		t.Errorf("Keys() returned %d items; want %d", i, len(expectedOrder))
	}
}

func TestOrderedMap_Values(t *testing.T) {
	om := cfgldr.NewOrderedMap[string, int]()

	// Test empty
	count := 0
	for v := range om.Values() {
		count++
		t.Errorf("unexpected value on empty map: %d", v)
	}
	if count != 0 {
		t.Errorf("empty GetValues() count = %d; want 0", count)
	}

	// Set up test data
	keys := []string{"a", "b", "c"}
	expectedValues := []int{10, 20, 30}
	for i, key := range keys {
		om.Set(key, expectedValues[i])
	}

	// Test GetValues iterator
	i := 0
	for v := range om.Values() {
		if i >= len(expectedValues) {
			t.Errorf("GetValues() returned more items than expected")
			break
		}
		if v != expectedValues[i] {
			t.Errorf("GetValues()[%d] = %d; want %d", i, v, expectedValues[i])
		}
		i++
	}
	if i != len(expectedValues) {
		t.Errorf("GetValues() returned %d items; want %d", i, len(expectedValues))
	}
}

func TestOrderedMap_GetKeys(t *testing.T) {
	om := cfgldr.NewOrderedMap[string, int]()

	// Test empty
	keys := om.GetKeys()
	if len(keys) != 0 {
		t.Errorf("GetKeys() on empty map length = %d; want 0", len(keys))
	}

	// Set up test data
	expectedKeys := []string{"x", "y", "z"}
	for i, key := range expectedKeys {
		om.Set(key, i)
	}

	// Test GetKeys
	keys = om.GetKeys()
	if len(keys) != len(expectedKeys) {
		t.Fatalf("GetKeys() length = %d; want %d", len(keys), len(expectedKeys))
	}
	for i, key := range keys {
		if key != expectedKeys[i] {
			t.Errorf("GetKeys()[%d] = %s; want %s", i, key, expectedKeys[i])
		}
	}

	// Verify returned slice is a copy (mutating it shouldn't affect the map)
	keys[0] = "modified"
	originalKeys := om.GetKeys()
	if originalKeys[0] == "modified" {
		t.Error("GetKeys() did not return a copy; original was modified")
	}
}

func TestOrderedMap_GetValues(t *testing.T) {
	om := cfgldr.NewOrderedMap[string, int]()

	// Test empty
	values := om.GetValues()
	if len(values) != 0 {
		t.Errorf("GetValues() on empty map length = %d; want 0", len(values))
	}

	// Set up test data
	keys := []string{"p", "q", "r"}
	expectedValues := []int{100, 200, 300}
	for i, key := range keys {
		om.Set(key, expectedValues[i])
	}

	// Test GetValues
	values = om.GetValues()
	if len(values) != len(expectedValues) {
		t.Fatalf("GetValues() length = %d; want %d", len(values), len(expectedValues))
	}
	for i, value := range values {
		if value != expectedValues[i] {
			t.Errorf("GetValues()[%d] = %d; want %d", i, value, expectedValues[i])
		}
	}
}

func TestOrderedMap_String(t *testing.T) {
	om := cfgldr.NewOrderedMap[string, int]()

	// Test empty
	str := om.String()
	if str != "" {
		t.Errorf("String() on empty map = %q; want empty string", str)
	}

	// Test single item
	om.Set("key", 42)
	str = om.String()
	expected := "key=42"
	if str != expected {
		t.Errorf("String() with single item = %q; want %q", str, expected)
	}

	// Test multiple items
	om.Set("another", 99)
	str = om.String()
	expected = "key=42 another=99"
	if str != expected {
		t.Errorf("String() with multiple items = %q; want %q", str, expected)
	}
}

func TestOrderedMap_IteratorSnapshot(t *testing.T) {
	om := cfgldr.NewOrderedMap[string, int]()

	// Set up initial data
	om.Set("a", 1)
	om.Set("b", 2)

	// Start iteration but don't complete it
	iter := om.Iterator()

	// Modify map during iteration
	om.Set("c", 3)
	om.Delete("a")

	// Complete iteration - should use snapshot from when iterator was created
	var collected []string
	for k := range iter {
		collected = append(collected, k)
	}

	// Should still see original state
	expected := []string{"a", "b"}
	if len(collected) != len(expected) {
		t.Fatalf("iterator snapshot length = %d; want %d", len(collected), len(expected))
	}
	for i, key := range collected {
		if key != expected[i] {
			t.Errorf("iterator snapshot[%d] = %s; want %s", i, key, expected[i])
		}
	}
}
