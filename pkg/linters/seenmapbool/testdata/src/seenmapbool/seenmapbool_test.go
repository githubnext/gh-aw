package seenmapbool

import "testing"

// TestSubtestSeenMapSkipped mirrors compiled_lock_files_test.go: a
// map[string]bool set declared inside a t.Run(...) closure in a _test.go
// file must not be reported, since the FuncLit is nested inside a test file
// that should be exempt from analysis (no `want` comment expected below).
func TestSubtestSeenMapSkipped(t *testing.T) {
	tests := []string{"a", "b", "a"}
	t.Run("subtest", func(t *testing.T) {
		seen := map[string]bool{}
		for _, name := range tests {
			seen[name] = true
		}
		_ = seen
	})
}
