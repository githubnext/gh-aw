package uncheckedtypeassertion

import "fmt"

// Good: two-value type assertion is safe.
func GoodTwoValue(v interface{}) {
	s, ok := v.(string)
	if ok {
		fmt.Println(s)
	}
}

// Good: type switch is safe — not flagged.
func GoodTypeSwitch(v interface{}) {
	switch t := v.(type) {
	case string:
		fmt.Println(t)
	}
}

// Bad: single-value assertion may panic.
func BadSingleValue(v interface{}) string {
	return v.(string) // want `type assertion x\.\(string\) is unchecked and may panic`
}

// Bad: single-value assertion stored in variable.
func BadSingleValueAssign(v interface{}) {
	s := v.(string) // want `type assertion x\.\(string\) is unchecked and may panic`
	fmt.Println(s)
}

// Good: two-value form with blank ok is still two-value.
func GoodTwoValueBlankOk(v interface{}) string {
	s, _ := v.(string)
	return s
}

// Good: two-value var declaration is safe.
func GoodTwoValueVarDecl(v interface{}) {
	var s, ok = v.(string)
	if ok {
		fmt.Println(s)
	}
}

// Good: parenthesized two-value assignment is safe.
func GoodTwoValueParen(v interface{}) {
	s, ok := (v.(string))
	if ok {
		fmt.Println(s)
	}
}

// Good: parenthesized two-value re-assignment is safe.
func GoodTwoValueParenAssign(v interface{}) {
	var s string
	var ok bool
	s, ok = (v.(string))
	if ok {
		fmt.Println(s)
	}
}

// Good: parenthesized two-value var declaration is safe.
func GoodTwoValueVarDeclParen(v interface{}) {
	var s, ok = (v.(string))
	if ok {
		fmt.Println(s)
	}
}

// Good: double-parenthesized two-value assignment is safe.
func GoodTwoValueDoubleParen(v interface{}) {
	s, ok := ((v.(string)))
	if ok {
		fmt.Println(s)
	}
}

// Bad: single-value var declaration may panic.
func BadSingleValueVarDecl(v interface{}) {
	var s = v.(string) // want `type assertion x\.\(string\) is unchecked and may panic`
	fmt.Println(s)
}

func SuppressedPreviousLine(v interface{}) string {
	//nolint:uncheckedtypeassertion
	return v.(string)
}

func SuppressedSameLine(v interface{}) string {
	return v.(string) //nolint:uncheckedtypeassertion
}
