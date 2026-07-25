package main

import (
	"fmt"
	"runtime/debug"
)

func compileWorkflowPanicError(r any) error {
	return fmt.Errorf("compileWorkflow panic: %v\n%s", r, debug.Stack())
}

func runCompileWithRecovery(doCompile func() (any, error), resolve func(any), reject func(error)) {
	defer func() {
		if r := recover(); r != nil {
			reject(compileWorkflowPanicError(r))
		}
	}()

	result, err := doCompile()
	if err != nil {
		reject(err)
		return
	}
	resolve(result)
}
