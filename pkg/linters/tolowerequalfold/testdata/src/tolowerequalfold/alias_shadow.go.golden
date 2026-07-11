package tolowerequalfold

import str "strings"

func aliasImportExamples() {
	a := "Alice"
	b := "alice"

	_ = str.ToLower(a) == str.ToLower(b)
	_ = str.ToUpper(a) == str.ToUpper(b)
}

func aliasImportTrackedExamples() {
	a := "Alice"
	b := "alice"

	x := str.ToLower(a)
	_ = x == "alice" // want `use strings\.EqualFold`

	y := str.ToUpper(b)
	_ = "ALICE" == y // want `use strings\.EqualFold`
}

func aliasImportMismatchedExamples() {
	a := "Alice"
	b := "Bob"

	// Alias with case-mismatched literal — must not be rewritten to EqualFold.
	x := str.ToLower(a)
	_ = x == "ALICE"

	y := str.ToUpper(a)
	_ = "alice" == y

	// Alias-vs-alias with mismatched conversion functions must not be rewritten.
	lower := str.ToLower(a)
	upper := str.ToUpper(b)
	_ = lower == upper
}

type shadowStrings struct{}

func (shadowStrings) ToLower(s string) string {
	return s
}

func shadowedIdentifierExample() {
	strings := shadowStrings{}
	a := "Alice"
	b := "alice"

	_ = strings.ToLower(a) == strings.ToLower(b)
}
