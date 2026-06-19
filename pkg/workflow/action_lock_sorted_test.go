//go:build !integration

package workflow

import (
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"testing"
)

func TestActionsLockJSONFieldsAreSorted(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		path string
	}{
		{
			name: "root actions lock",
			path: filepath.Join("..", "..", ".github", "aw", "actions-lock.json"),
		},
		{
			name: "pkg workflow actions lock",
			path: filepath.Join(".github", "aw", "actions-lock.json"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			data, err := os.ReadFile(tc.path)
			if err != nil {
				t.Fatalf("read %s: %v", tc.path, err)
			}

			var payload struct {
				Entries    map[string]json.RawMessage `json:"entries"`
				Containers map[string]json.RawMessage `json:"containers"`
			}

			if err := json.Unmarshal(data, &payload); err != nil {
				t.Fatalf("parse %s: %v", tc.path, err)
			}

			assertMapKeysSortedInFileOrder(t, tc.path, data, "entries", payload.Entries)
			assertMapKeysSortedInFileOrder(t, tc.path, data, "containers", payload.Containers)
		})
	}
}

func assertMapKeysSortedInFileOrder(t *testing.T, path string, content []byte, field string, m map[string]json.RawMessage) {
	t.Helper()

	if len(m) == 0 {
		return
	}

	inOrder := make([]string, 0, len(m))
	for k := range m {
		inOrder = append(inOrder, k)
	}

	sorted := slices.Clone(inOrder)
	slices.Sort(sorted)

	// map iteration order is random; rebuild order from file by scanning sorted keys by first appearance.
	slices.SortFunc(inOrder, func(a, b string) int {
		qa := []byte(`"` + a + `"`)
		qb := []byte(`"` + b + `"`)
		ia := indexOfBytes(content, qa)
		ib := indexOfBytes(content, qb)
		switch {
		case ia < ib:
			return -1
		case ia > ib:
			return 1
		default:
			return 0
		}
	})

	for i := range sorted {
		if sorted[i] != inOrder[i] {
			t.Fatalf("%s field %q is not sorted lexicographically", path, field)
		}
	}
}

func indexOfBytes(haystack []byte, needle []byte) int {
	if len(needle) == 0 || len(haystack) < len(needle) {
		return -1
	}
	for i := 0; i <= len(haystack)-len(needle); i++ {
		if string(haystack[i:i+len(needle)]) == string(needle) {
			return i
		}
	}
	return -1
}
