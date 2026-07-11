package tolowerequalfold

import str "strings"

func aliasImportExamples() {
	a := "Alice"
	b := "alice"

	_ = str.ToLower(a) == str.ToLower(b) // want `use strings\.EqualFold`
	_ = str.ToUpper(a) == str.ToUpper(b) // want `use strings\.EqualFold`
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

	// Alias with case-mismatched literal — must not be rewritten to EqualFold.
	x := str.ToLower(a)
	_ = x == "ALICE"

	y := str.ToUpper(a)
	_ = "alice" == y
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
