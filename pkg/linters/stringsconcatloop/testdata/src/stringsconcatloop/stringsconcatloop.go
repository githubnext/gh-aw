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

	// x = x + y form in a range loop – should be flagged.
	accum := ""
	for _, p := range parts {
		accum = accum + p // want `string concatenation with \+= inside a loop`
	}
	_ = accum

	// x = x + y form in a classic for loop – should be flagged.
	s2 := ""
	for i := 0; i < len(parts); i++ {
		s2 = s2 + parts[i] // want `string concatenation with \+= inside a loop`
	}
	_ = s2
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

	// x = x + y outside any loop – not flagged.
	outside := "prefix"
	outside = outside + "suffix"
	_ = outside

	// x = y + x (left operand is not the LHS) – not flagged.
	accum2 := ""
	for _, p := range parts {
		accum2 = p + accum2
	}
	_ = accum2

	// Range value variable reassigned per iteration – not a cross-iteration
	// accumulator, so not flagged.
	for _, line := range parts {
		line = line + " suffix"
		_ = line
	}

	// Range key variable over a string-keyed map – not a cross-iteration
	// accumulator, so not flagged.
	m := map[string]int{"a": 1, "b": 2}
	for k := range m {
		k = k + "_x"
		_ = k
	}

	// x = x + y inside a func literal inside a loop – not flagged. The linter
	// intentionally stops at func literal boundaries.
	accum3 := ""
	for _, p := range parts {
		func() {
			accum3 = accum3 + p
		}()
	}
	_ = accum3
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
