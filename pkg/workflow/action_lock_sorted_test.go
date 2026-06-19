//go:build !integration

package workflow

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
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

	inOrder, err := extractObjectKeyOrder(content, field)
	if err != nil {
		t.Fatalf("extract key order for %s field %q: %v", path, field, err)
	}

	sorted := slices.Clone(inOrder)
	slices.Sort(sorted)

	for i := range sorted {
		if sorted[i] != inOrder[i] {
			t.Fatalf("%s field %q is not sorted lexicographically", path, field)
		}
	}
}

func extractObjectKeyOrder(content []byte, field string) ([]string, error) {
	dec := json.NewDecoder(bytes.NewReader(content))

	start, err := dec.Token()
	if err != nil {
		return nil, err
	}
	if d, ok := start.(json.Delim); !ok || d != '{' {
		return nil, errors.New("root is not an object")
	}

	for dec.More() {
		keyToken, err := dec.Token()
		if err != nil {
			return nil, err
		}
		key, ok := keyToken.(string)
		if !ok {
			return nil, fmt.Errorf("non-string root key token %T", keyToken)
		}

		if key != field {
			var ignored json.RawMessage
			if err := dec.Decode(&ignored); err != nil {
				return nil, err
			}
			continue
		}

		objectStart, err := dec.Token()
		if err != nil {
			return nil, err
		}
		if d, ok := objectStart.(json.Delim); !ok || d != '{' {
			return nil, fmt.Errorf("field %q is not an object", field)
		}

		keys := make([]string, 0, 8)
		for dec.More() {
			memberKeyToken, err := dec.Token()
			if err != nil {
				return nil, err
			}
			memberKey, ok := memberKeyToken.(string)
			if !ok {
				return nil, fmt.Errorf("non-string member key token %T", memberKeyToken)
			}
			keys = append(keys, memberKey)

			var ignored json.RawMessage
			if err := dec.Decode(&ignored); err != nil {
				return nil, err
			}
		}

		objectEnd, err := dec.Token()
		if err != nil {
			return nil, err
		}
		if d, ok := objectEnd.(json.Delim); !ok || d != '}' {
			return nil, fmt.Errorf("field %q object not closed", field)
		}
		return keys, nil
	}

	return nil, fmt.Errorf("field %q not found", field)
}
