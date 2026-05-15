// Package a is the test fixture for the excessivefuncparams analyzer.
package a

// fewParams is within the default limit — no diagnostic expected.
func fewParams(a, b, c int) {}

// tooManyParams has more than 8 parameters and should be flagged.
func tooManyParams(a, b, c, d, e, f, g, h, i int) { // want `tooManyParams has 9 parameters \(limit: 8\); consider using an options struct`
}
