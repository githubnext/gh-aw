// Package fmterrorfnoverbs contains test fixtures for the fmterrorfnoverbs linter.
package fmterrorfnoverbs

import (
	"errors"
	"fmt"
)

func bad() error {
	return fmt.Errorf("something went wrong") // want `fmt\.Errorf called with no format verbs; use errors\.New`
}

func badMultiLine() error {
	return fmt.Errorf("interactive input not available in Wasm") // want `fmt\.Errorf called with no format verbs; use errors\.New`
}

func good() error {
	return errors.New("something went wrong")
}

func goodWithVerb(name string) error {
	return fmt.Errorf("failed to process %s", name)
}

func goodWithWrap(err error) error {
	return fmt.Errorf("wrapper: %w", err)
}
