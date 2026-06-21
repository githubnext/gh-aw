package tolowerequalfold

import str "strings"

func aliasImportExamples() {
	a := "Alice"
	b := "alice"

	_ = str.ToLower(a) == str.ToLower(b) // want `use strings\.EqualFold`
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
