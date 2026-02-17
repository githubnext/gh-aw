//go:build js && wasm

package main

import (
	"syscall/js"

	"github.com/github/gh-aw/pkg/workflow"
)

func main() {
	js.Global().Set("compileWorkflow", js.FuncOf(compileWorkflow))
	select {}
}

// compileWorkflow is the JS-callable function.
// Usage: compileWorkflow(markdownString) → Promise<{yaml, warnings, error}>
//
// Only a single argument (the markdown string) is accepted.
// Import resolution is not currently supported in the Wasm build.
func compileWorkflow(this js.Value, args []js.Value) any {
	if len(args) < 1 {
		return newRejectedPromise("compileWorkflow requires exactly 1 argument: markdown string")
	}

	if len(args) > 1 {
		return newRejectedPromise("compileWorkflow accepts only 1 argument; importResolver is not supported in the Wasm build")
	}

	markdown := args[0].String()

	var handler js.Func
	handler = js.FuncOf(func(this js.Value, promiseArgs []js.Value) any {
		resolve := promiseArgs[0]
		reject := promiseArgs[1]

		go func() {
			defer handler.Release()

			result, err := doCompile(markdown)
			if err != nil {
				reject.Invoke(js.Global().Get("Error").New(err.Error()))
				return
			}
			resolve.Invoke(result)
		}()

		return nil
	})

	return js.Global().Get("Promise").New(handler)
}

// doCompile performs the actual compilation entirely in memory — no filesystem access.
func doCompile(markdown string) (js.Value, error) {
	compiler := workflow.NewCompiler(
		workflow.WithNoEmit(true),
		workflow.WithSkipValidation(true),
	)

	// Parse directly from string — no temp files needed
	workflowData, err := compiler.ParseWorkflowString(markdown, "workflow.md")
	if err != nil {
		return js.Undefined(), err
	}

	yamlContent, err := compiler.CompileToYAML(workflowData, "workflow.md")
	if err != nil {
		return js.Undefined(), err
	}

	result := js.Global().Get("Object").New()
	result.Set("yaml", yamlContent)
	result.Set("error", js.Null())

	warnings := js.Global().Get("Array").New()
	result.Set("warnings", warnings)

	return result, nil
}

func newRejectedPromise(msg string) js.Value {
	var handler js.Func
	handler = js.FuncOf(func(this js.Value, args []js.Value) any {
		defer handler.Release()
		reject := args[1]
		reject.Invoke(js.Global().Get("Error").New(msg))
		return nil
	})
	return js.Global().Get("Promise").New(handler)
}
