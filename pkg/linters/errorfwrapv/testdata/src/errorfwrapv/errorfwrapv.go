package errorfwrapv

import (
"errors"
"fmt"
)

var ErrBase = errors.New("base error")

// BadVWrap uses %v to format an error — should be %w.
func BadVWrap(err error) error {
return fmt.Errorf("operation failed: %v", err) // want `fmt\.Errorf formats an error argument with %v`
}

// BadVWrapExtra has %v for an error arg alongside a non-error arg.
func BadVWrapExtra(err error, count int) error {
return fmt.Errorf("error %v occurred %d times", err, count) // want `fmt\.Errorf formats an error argument with %v`
}

// GoodWWrap uses %w to properly wrap the error.
func GoodWWrap(err error) error {
return fmt.Errorf("operation failed: %w", err)
}

// GoodNonErrorVerb uses %v for a non-error argument only.
func GoodNonErrorVerb(name string) error {
return fmt.Errorf("operation %v failed", name)
}

// GoodMixedVerbs uses %w for the error and %v for a non-error.
func GoodMixedVerbs(name string, err error) error {
return fmt.Errorf("operation %v failed: %w", name, err)
}
