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
