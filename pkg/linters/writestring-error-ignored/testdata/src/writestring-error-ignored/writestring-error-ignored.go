package writestringerrorignored

import (
	"strings"
)

// badBareWriteString: direct call with error ignored
func badBareWriteString() {
	var b strings.Builder
	b.WriteString("hello") // want `error from WriteString is ignored`
}

// badBareWriteByte: direct call with error ignored
func badBareWriteByte() {
	var b strings.Builder
	b.WriteByte('x') // want `error from WriteByte is ignored`
}

// badBareWriteRune: direct call with error ignored
func badBareWriteRune() {
	var b strings.Builder
	b.WriteRune('Ω') // want `error from WriteRune is ignored`
}

// badAssignBlankWriteString: blank assignment with error discarded
func badAssignBlankWriteString() {
	var b strings.Builder
	_, _ = b.WriteString("hello") // want `error from WriteString is ignored`
}

// badAssignBlankWriteByte: blank assignment with error discarded
func badAssignBlankWriteByte() {
	var b strings.Builder
	_ = b.WriteByte('x') // want `error from WriteByte is ignored`
}

// badAssignBlankWriteRune: blank assignment with error discarded
func badAssignBlankWriteRune() {
	var b strings.Builder
	_, _ = b.WriteRune('x') // want `error from WriteRune is ignored`
}

// goodNamedAssignment: error assigned to named variable
func goodNamedAssignment() {
	var b strings.Builder
	n, err := b.WriteString("hello")
	if err != nil {
		_ = err
	}
	_ = n
}

// goodErrorCheck: error is checked
func goodErrorCheck() {
	var b strings.Builder
	if _, err := b.WriteString("hello"); err != nil {
		_ = err
	}
}

// goodOnlyResultAssigned: only result assigned, not error
func goodOnlyResultAssigned() {
	var b strings.Builder
	n, _ := b.WriteString("hello")
	_ = n
}

// goodNamedBytesAssignment: error assigned to named variable (WriteByte)
func goodNamedBytesAssignment() {
	var b strings.Builder
	err := b.WriteByte('x')
	if err != nil {
		_ = err
	}
}

// goodDeferredCall: deferred calls should not be flagged
func goodDeferredCall() {
	var b strings.Builder
	defer b.WriteString("cleanup")
}

// goodDeferredWriteByte: deferred byte write should not be flagged
func goodDeferredWriteByte() {
	var b strings.Builder
	defer b.WriteByte('\n')
}

// goodDeferredWriteRune: deferred rune write should not be flagged
func goodDeferredWriteRune() {
	var b strings.Builder
	defer b.WriteRune('x')
}
