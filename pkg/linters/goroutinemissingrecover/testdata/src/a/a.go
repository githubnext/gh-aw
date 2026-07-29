// Package a is the test fixture for the goroutinemissingrecover analyzer.
package a

// safeGoroutine has a top-level defer/recover — no diagnostic expected.
func safeGoroutine() {
	go func() {
		defer func() {
			if r := recover(); r != nil {
				_ = r
			}
		}()
		panic("oops")
	}()
}

// unsafeGoroutine has no recover — should be flagged.
func unsafeGoroutine() {
	go func() { // want `goroutine launched via a function literal without a top-level defer/recover`
		panic("oops")
	}()
}

// namedFuncGoroutine calls a named function — out of scope, not flagged.
func namedFuncHelper() {}

func namedFuncGoroutine() {
	go namedFuncHelper()
}

// unrelatedDeferGoroutine has a defer but no recover — should be flagged.
func unrelatedDeferGoroutine() {
	go func() { // want `goroutine launched via a function literal without a top-level defer/recover`
		defer func() {
			_ = 1
		}()
		panic("oops")
	}()
}

// suppressedGoroutine is suppressed via nolint — no diagnostic expected.
func suppressedGoroutine() {
	//nolint:goroutinemissingrecover
	go func() {
		panic("oops")
	}()
}

// parenthesizedGoroutine uses the parenthesised literal syntax — should be flagged.
func parenthesizedGoroutine() {
	go (func() { // want `goroutine launched via a function literal without a top-level defer/recover`
		panic("oops")
	})()
}

// nestedClosureRecoverGoroutine has recover() buried in a nested closure inside the
// defer literal. recover() only stops a panic in the same stack frame, so the
// nested recover does NOT protect the goroutine — should be flagged.
func nestedClosureRecoverGoroutine() {
	go func() { // want `goroutine launched via a function literal without a top-level defer/recover`
		defer func() {
			// recover() is inside a nested closure — it only guards the nested call,
			// not the outer goroutine.
			func() { recover() }()
		}()
		panic("oops")
	}()
}

// nestedRecoverOnlyGoroutine — outer goroutine has no recover; inner func does.
// The outer goroutine is unprotected — should be flagged.
func nestedRecoverOnlyGoroutine() {
	go func() { // want `goroutine launched via a function literal without a top-level defer/recover`
		func() {
			defer func() {
				if r := recover(); r != nil {
					_ = r
				}
			}()
			panic("inner")
		}()
		panic("outer") // this panic is unrecovered
	}()
}

