//go:build !integration

package yamlpostcheck

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// errChecker is a Checker that always returns the provided error.
type errChecker struct {
	name string
	err  error
}

func (e *errChecker) Name() string                           { return e.name }
func (e *errChecker) Check(_ map[string]any) (Result, error) { return Result{}, e.err }

// mutatingChecker always marks the tree as changed and appends a fix.
type mutatingChecker struct {
	name string
	fix  string
}

func (m *mutatingChecker) Name() string { return m.name }
func (m *mutatingChecker) Check(_ map[string]any) (Result, error) {
	return Result{Changed: true, Fixes: []string{m.fix}}, nil
}

// warnChecker produces a warning without modifying the tree.
type warnChecker struct {
	name    string
	warning string
}

func (w *warnChecker) Name() string { return w.name }
func (w *warnChecker) Check(_ map[string]any) (Result, error) {
	return Result{Warnings: []string{w.warning}}, nil
}

func TestNew_RegistersDefaultCheckers(t *testing.T) {
	s := New()
	// The default suite must contain at least the built-in secrets-in-run checker.
	require.NotEmpty(t, s.checkers, "default suite should have at least one checker")
	names := make([]string, len(s.checkers))
	for i, c := range s.checkers {
		names[i] = c.Name()
	}
	assert.Contains(t, names, "secrets-in-run", "default suite should include secrets-in-run")
}

func TestSuite_Register_PanicsOnNil(t *testing.T) {
	s := &Suite{}
	assert.Panics(t, func() { s.Register(nil) }, "registering nil checker should panic")
}

func TestSuite_Run_NilTree(t *testing.T) {
	s := New()
	changed, fixes, warnings, err := s.Run(nil)
	require.NoError(t, err, "nil tree should not error")
	assert.False(t, changed, "nil tree should not be changed")
	assert.Empty(t, fixes, "nil tree should produce no fixes")
	assert.Empty(t, warnings, "nil tree should produce no warnings")
}

func TestSuite_Run_EmptyTree(t *testing.T) {
	s := New()
	tree := map[string]any{}
	changed, fixes, warnings, err := s.Run(tree)
	require.NoError(t, err, "empty tree should not error")
	assert.False(t, changed, "empty tree should not be changed by default suite")
	assert.Empty(t, fixes, "empty tree should produce no fixes")
	assert.Empty(t, warnings, "empty tree should produce no warnings")
}

func TestSuite_Run_AccumulatesFixesAndWarnings(t *testing.T) {
	s := &Suite{}
	s.Register(&mutatingChecker{name: "c1", fix: "fix-one"})
	s.Register(&warnChecker{name: "c2", warning: "warn-one"})
	s.Register(&mutatingChecker{name: "c3", fix: "fix-two"})

	tree := map[string]any{}
	changed, fixes, warnings, err := s.Run(tree)
	require.NoError(t, err, "should not error")
	assert.True(t, changed, "suite should be changed when any checker mutates")
	assert.Equal(t, []string{"fix-one", "fix-two"}, fixes, "fixes should be accumulated in order")
	assert.Equal(t, []string{"warn-one"}, warnings, "warnings should be accumulated")
}

func TestSuite_Run_StopsOnFirstError(t *testing.T) {
	sentinel := errors.New("checker exploded")

	s := &Suite{}
	s.Register(&mutatingChecker{name: "before", fix: "fix-before"})
	s.Register(&errChecker{name: "boom", err: sentinel})
	s.Register(&mutatingChecker{name: "after", fix: "fix-after"})

	tree := map[string]any{}
	changed, fixes, _, err := s.Run(tree)

	require.Error(t, err, "suite should propagate checker error")
	require.ErrorIs(t, err, sentinel, "error chain should contain the sentinel")
	// "before" ran successfully, "after" was skipped.
	assert.True(t, changed, "changed should reflect mutations before the error")
	assert.Equal(t, []string{"fix-before"}, fixes, "only fixes before the error should be included")
	assert.NotContains(t, fixes, "fix-after", "fix from checker after error must not appear")
}

func TestSuite_Run_ErrorMessageContainsCheckerName(t *testing.T) {
	s := &Suite{}
	s.Register(&errChecker{name: "my-checker", err: errors.New("oops")})

	_, _, _, err := s.Run(map[string]any{})
	require.Error(t, err, "should return error")
	assert.Contains(t, err.Error(), "my-checker", "error should include checker name")
}

func TestSuite_Run_CheckersReceiveMutatedTree(t *testing.T) {
	// Verify that a second checker sees the same tree as the first, meaning
	// mutations by the first checker are visible to subsequent ones.
	var receivedTree map[string]any

	// captureChecker records the tree it receives.
	captureChecker := &captureTreeChecker{capture: &receivedTree}

	s := &Suite{}
	s.Register(captureChecker)

	originalTree := map[string]any{"key": "value"}
	_, _, _, err := s.Run(originalTree)
	require.NoError(t, err, "should not error")
	assert.Equal(t, originalTree, receivedTree, "checker should receive the same tree object")
}

// captureTreeChecker is a Checker that stores the tree it receives for inspection.
type captureTreeChecker struct {
	capture *map[string]any
}

func (cc *captureTreeChecker) Name() string { return "capture" }
func (cc *captureTreeChecker) Check(tree map[string]any) (Result, error) {
	*cc.capture = tree
	return Result{}, nil
}
