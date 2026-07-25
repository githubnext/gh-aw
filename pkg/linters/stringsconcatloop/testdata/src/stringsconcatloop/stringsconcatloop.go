package stringsconcatloop

import "strings"

func bad() {
	parts := []string{"a", "b", "c"}

	// Basic range loop – should be flagged.
	result := ""
	for _, p := range parts {
		result += p // want `string concatenation with \+= inside a loop`
	}
	_ = result

	// Classic for loop – should be flagged.
	s := ""
	for i := 0; i < len(parts); i++ {
		s += parts[i] // want `string concatenation with \+= inside a loop`
	}
	_ = s

	// Named string type – should also be flagged.
	type myString string
	var ms myString
	for _, p := range parts {
		ms += myString(p) // want `string concatenation with \+= inside a loop`
	}
	_ = ms
}

func good() {
	parts := []string{"a", "b", "c"}

	// Using strings.Builder – not flagged.
	var sb strings.Builder
	for _, p := range parts {
		sb.WriteString(p)
	}
	_ = sb.String()

	// += outside any loop – not flagged.
	result := "prefix"
	result += "suffix"
	_ = result

	// Integer += inside a loop – not flagged.
	n := 0
	for i := range parts {
		n += i
	}
	_ = n

	// String += inside a func literal inside a loop – not flagged. The linter
	// intentionally stops at func literal boundaries.
	acc := ""
	for _, p := range parts {
		func() {
			acc += p
		}()
	}
	_ = acc
}

func nolintDirective() {
	parts := []string{"a", "b", "c"}

	result := ""
	for _, p := range parts { //nolint:stringsconcatloop
		result += p
	}
	_ = result

	result2 := ""
	for _, p := range parts { //nolint:stringsconcatloop
		_ = strings.TrimSpace(p)
		result2 += p
	}
	_ = result2
}
