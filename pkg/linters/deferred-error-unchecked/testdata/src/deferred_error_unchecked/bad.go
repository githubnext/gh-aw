package deferred_error_unchecked

import (
	"fmt"
	"io"
	"os"
)

// Bad case: deferred call that returns error (not assigned, not checked)
func badDeferClose(filename string) error {
	f, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer f.Close() // want "deferred call discards error return value"
	return nil
}

// Bad case: deferred method call on io.Closer
func badDeferIOCloser(c io.Closer) {
	defer c.Close() // want "deferred call discards error return value"
}

// Bad case: function that returns error
func errorReturningFunc() error {
	return fmt.Errorf("some error")
}

func badDeferErrorFunc() {
	defer errorReturningFunc() // want "deferred call discards error return value"
}

// Bad case: method that returns error
type ErrorCloser struct {
	closed bool
}

func (e *ErrorCloser) Close() error {
	e.closed = true
	return nil
}

func badDeferMethodError() {
	e := &ErrorCloser{}
	defer e.Close() // want "deferred call discards error return value"
}

// Bad case: interface method returning error
type GenericCloser interface {
	Close() error
}

func badDeferInterface(c GenericCloser) {
	defer c.Close() // want "deferred call discards error return value"
}

// Bad case: nested call returning error
func nestedErrorReturner() error {
	return nil
}

func badDeferNested() {
	defer nestedErrorReturner() // want "deferred call discards error return value"
}
