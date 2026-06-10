// Package tolowerequalfold contains test fixtures for the tolowerequalfold linter.
package tolowerequalfold

import "strings"

func flaggedExamples() {
	name := "Alice"

	// should use strings.EqualFold(name, "alice")
	_ = strings.ToLower(name) == "alice" // want `use strings\.EqualFold`
	_ = strings.ToUpper(name) == "ALICE" // want `use strings\.EqualFold`
	_ = "alice" == strings.ToLower(name) // want `use strings\.EqualFold`
	_ = strings.ToLower(name) != "alice" // want `use strings\.EqualFold`
	_ = strings.ToLower(name) == strings.ToLower("alice") // want `use strings\.EqualFold`

	lower := strings.ToLower(name)
	_ = lower == "alice" // want `use strings\.EqualFold`
}

func okExamples() {
	name := "Alice"

	// EqualFold is already idiomatic — no diagnostic expected
	_ = strings.EqualFold(name, "alice")

	// Regular case-sensitive comparison — no diagnostic
	_ = name == "Alice"
	_ = strings.ToLower(name) // used standalone, not in a comparison
	_ = strings.ToLower(name) == name
	_ = strings.ToLower(name) != name

	lower := strings.ToLower(name)
	_ = lower == name
}

func suppressedExamples() {
	name := "Alice"
	_ = strings.ToLower(name) == "alice" //nolint:tolowerequalfold
}

