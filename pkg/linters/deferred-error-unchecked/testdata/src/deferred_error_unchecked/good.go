package deferred_error_unchecked

import (
	"fmt"
	"io"
	"os"
)

// Good case: error is checked in a deferred closure
func goodCheckErrorInClosure(filename string) error {
	f, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer func() {
		if err := f.Close(); err != nil {
			fmt.Fprintf(os.Stderr, "close error: %v\n", err)
		}
	}()
	return nil
}

// Good case: error is assigned (not ignored)
func goodAssignError() {
	var c io.Closer
	defer func() {
		_ = c.Close() // Assigning to _, but explicitly done
	}()
}

// Good case: function returns no error
type NoErrorReturner struct{}

func (n *NoErrorReturner) Cleanup() {}

func goodNoError() {
	n := &NoErrorReturner{}
	defer n.Cleanup() // OK: Cleanup() returns nothing
}

// Good case: explicit error assignment
func goodExplicitAssignment() {
	var f *os.File
	err := f.Close()
	defer func() {
		if err != nil {
			// handle error
		}
	}()
}

// Good case: multiple return values, not ending in error
func multiReturnNoError() (int, string) {
	return 1, "hello"
}

func goodMultiReturn() {
	defer func() {
		a, b := multiReturnNoError()
		_ = a
		_ = b
	}()
}
