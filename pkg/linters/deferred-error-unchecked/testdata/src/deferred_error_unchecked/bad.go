package deferred_error_unchecked

import (
	"fmt"
	"io"
	"os"
)

// Direct Close calls are covered only by unchecked_deferredclose.
func badDeferClose(filename string) error {
	f, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer f.Close()
	return nil
}

// Direct Close calls on interfaces also belong to unchecked_deferredclose.
func badDeferIOCloser(c io.Closer) {
	defer c.Close()
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
	defer e.Close()
}

// Bad case: interface method returning error
type GenericCloser interface {
	Close() error
}

func badDeferInterface(c GenericCloser) {
	defer c.Close()
}

// Bad case: nested call returning error
func nestedErrorReturner() error {
	return nil
}

func badDeferNested() {
	defer nestedErrorReturner() // want "deferred call discards error return value"
}

func (e *ErrorCloser) Cleanup() error { return nil }

func badOtherMethod(e *ErrorCloser) {
	defer e.Cleanup() // want "deferred call discards error return value"
}

type ArgumentCloser struct{}

func (*ArgumentCloser) Close(int) error { return nil }

func badCloseWithArgument(c *ArgumentCloser) {
	defer c.Close(1) // want "deferred call discards error return value"
}

func parenthesizedClose(c io.Closer) {
	defer (c.Close)()
}

func suppressedClose(c io.Closer) {
	defer c.Close() //nolint:unchecked_deferredclose
}
