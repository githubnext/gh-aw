// Package fmterrorfnoverbs contains test fixtures for the fmterrorfnoverbs linter.
package fmterrorfnoverbs

import (
	"errors"
	fmtalias "fmt"
)

func bad() error {
	return fmtalias.Errorf("something went wrong") // want `fmt\.Errorf called with no format verbs; use errors\.New`
}

func badMultiLine() error {
	return fmtalias.Errorf("interactive input not available in Wasm") // want `fmt\.Errorf called with no format verbs; use errors\.New`
}

func good() error {
	return errors.New("something went wrong")
}

func goodWithVerb(name string) error {
	return fmtalias.Errorf("failed to process %s", name)
}

func goodWithWrap(err error) error {
	return fmtalias.Errorf("wrapper: %w", err)
}

func suppressedPreviousLine() error {
	//nolint:fmterrorfnoverbs
	return fmtalias.Errorf("this is intentionally static")
}

func suppressedSameLine() error {
	return fmtalias.Errorf("this is intentionally static") //nolint:fmterrorfnoverbs
}

func badAliasedImport() error {
	return fmtalias.Errorf("alias import static message") // want `fmt\.Errorf called with no format verbs; use errors\.New`
}

type shadowFormatter struct{}

func (shadowFormatter) Errorf(msg string, _ ...any) error { return errors.New(msg) }

func localShadowNotFlagged() error {
	fmt := shadowFormatter{}
	return fmt.Errorf("not fmt package")
}
