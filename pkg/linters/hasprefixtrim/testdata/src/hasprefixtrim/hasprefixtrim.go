package hasprefixtrim

import "strings"

type T struct{ name string }

func bad(s string, p string, t *T) {
	if strings.HasPrefix(s, p) { // want `use strings.TrimPrefix\(s, p\)`
		s = s[len(p):]
	}
	if strings.HasPrefix(s, "foo/") { // want `use strings.TrimPrefix\(s, "foo/"\)`
		s = s[len("foo/"):]
	}
	if strings.HasPrefix(s, "foo/") { // want `use strings.TrimPrefix`
		s = s[4:]
	}
	if strings.HasSuffix(t.name, ".go") { // want `use strings.TrimSuffix\(t.name, ".go"\)`
		t.name = t.name[:len(t.name)-len(".go")]
	}
	if strings.HasSuffix(s, ".go") { // want `use strings.TrimSuffix`
		s = s[:len(s)-3]
	}
	if strings.HasSuffix(s, p) { // want `use strings.TrimSuffix`
		s = s[:len(s)-len(p)]
	}
	_ = s
}

func good(s, p, q string, t *T) {
	if strings.HasPrefix(s, p) {
		s = s[len(q):] // different length arg
	}
	if strings.HasPrefix(s, "ab") {
		s = s[3:] // wrong literal length
	}
	if strings.HasPrefix(s, p) {
		s = s[len(p):]
		t.name = s // extra statement
	}
	if strings.HasPrefix(s, p) {
		q = s[len(p):] // different target
	}
	if strings.HasPrefix(s, p) {
		s = s[len(p):]
	} else {
		s = "x"
	}
	if x := 1; strings.HasPrefix(s, p) && x > 0 {
		s = s[len(p):]
	}
	if strings.HasPrefix(s, p) && len(s) > 5 {
		s = s[len(p):]
	}
	if strings.HasSuffix(s, p) {
		s = s[:len(s)-len(q)]
	}
	if strings.HasPrefix(s, p) {
		s = s[len(p):len(s)]
	}
	if strings.HasPrefix(t.name, p) {
		s = t.name[len(p):]
	}
	s = strings.TrimPrefix(s, p)
}

func elseIf(s string) {
	if strings.HasPrefix(s, "a/") {
		s = "x"
	} else if strings.HasPrefix(s, "b/") {
		s = s[2:]
	}
	_ = s
}
