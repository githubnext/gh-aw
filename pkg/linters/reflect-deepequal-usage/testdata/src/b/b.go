// Package b is the test fixture for negative cases of the reflect-deepequal-usage analyzer.
package b

import (
	"reflect"
)

// testTypedEquality should NOT be flagged - using typed equality
func testTypedEquality(a, b int) bool {
	return a == b
}

// testStringEquality should NOT be flagged - comparing strings directly
func testStringEquality(a, b string) bool {
	return a == b
}

// testSliceEquality should NOT be flagged - comparing slices with other methods
func testSliceEquality(a, b []int) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// testMapEquality should NOT be flagged - comparing maps without reflect.DeepEqual
func testMapEquality(a, b map[string]int) bool {
	if len(a) != len(b) {
		return false
	}
	for k, v := range a {
		if b[k] != v {
			return false
		}
	}
	return true
}

// testReflectNotDeepEqual should NOT be flagged - using reflect but different function
func testReflectNotDeepEqual(t interface{}) bool {
	return reflect.TypeOf(t) != nil
}

// testReflectValueEqual should NOT be flagged - using reflect.Value.Equal
func testReflectValueEqual(a, b reflect.Value) bool {
	return a.Equal(b)
}

// testStructEquality should NOT be flagged - comparing structs with typed equality
func testStructEquality(a, b struct{ x int }) bool {
	return a == b
}

// testNilComparison should NOT be flagged - comparing with nil
func testNilComparison(a interface{}) bool {
	return a == nil
}

func testWithNolintDirective(a, b interface{}) bool {
	// This should NOT be flagged because of the nolint directive
	return reflect.DeepEqual(a, b) //nolint:reflectdeepequalusage
}

func unrelatedEqual(a, b interface{}) bool {
	return false
}

func testReassignedAlias(a, b interface{}) bool {
	equal := reflect.DeepEqual
	equal = unrelatedEqual
	return equal(a, b)
}

func testReassignedInShortDeclaration(a, b interface{}) bool {
	equal := reflect.DeepEqual
	equal, other := unrelatedEqual, 0
	_ = other
	return equal(a, b)
}

func testConditionalReassignment(a, b interface{}, replace bool) bool {
	equal := reflect.DeepEqual
	if replace {
		equal = unrelatedEqual
	}
	return equal(a, b)
}

// Mutable aliases are unresolved even before the write, without flow analysis.
func testLaterReassignment(a, b interface{}) bool {
	equal := reflect.DeepEqual
	result := equal(a, b)
	equal = unrelatedEqual
	return result || equal(a, b)
}

func testUninitializedAlias(a, b interface{}) bool {
	var equal func(interface{}, interface{}) bool
	equal = unrelatedEqual
	return equal(a, b)
}

func testClosureReassignment(a, b interface{}) bool {
	equal := reflect.DeepEqual
	func() { equal = unrelatedEqual }()
	return equal(a, b)
}

func testAddressEscape(a, b interface{}) bool {
	equal := reflect.DeepEqual
	ptr := &(equal)
	*ptr = unrelatedEqual
	return equal(a, b)
}

func testRangeReassignment(a, b interface{}) bool {
	equal := reflect.DeepEqual
	for _, equal = range []func(interface{}, interface{}) bool{unrelatedEqual} {
		return equal(a, b)
	}
	return false
}

func testMutableAliasChain(a, b interface{}) bool {
	equal := reflect.DeepEqual
	equal = unrelatedEqual
	alias := equal
	return alias(a, b)
}

func testShadowedAlias(a, b interface{}) bool {
	equal := reflect.DeepEqual
	_ = equal
	{
		equal := unrelatedEqual
		return equal(a, b)
	}
}

var packageEqual = reflect.DeepEqual

func testPackageAlias(a, b interface{}) bool {
	packageEqual = unrelatedEqual
	return packageEqual(a, b)
}

func testAliasWithNolint(a, b interface{}) bool {
	equal := reflect.DeepEqual
	return equal(a, b) //nolint:reflectdeepequalusage
}

func testUnrelatedFunctionParameter(equal func(interface{}, interface{}) bool, a, b interface{}) bool {
	return equal(a, b)
}

type comparer struct{}

func (comparer) DeepEqual(a, b interface{}) bool {
	return false
}

func testUnrelatedMethod(a, b interface{}) bool {
	reflect := comparer{}
	equal := reflect.DeepEqual
	return equal(a, b)
}
