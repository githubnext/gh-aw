//go:build js && wasm

package main

import (
	"strings"
	"syscall/js"

	"github.com/github/gh-aw/pkg/parser"
	"github.com/github/gh-aw/pkg/workflow"
)

// Build-time variables injected via -X linker flags (mirrors cmd/gh-aw/main.go).
var (
	version   = "dev"
	isRelease = "false"
)

func main() {
	workflow.SetVersion(version)
	workflow.SetIsRelease(isRelease == "true")

	js.Global().Set("compileWorkflow", js.FuncOf(compileWorkflow))
	js.Global().Set("getVersion", js.FuncOf(getVersionFunc))
	select {}
}

// compileWorkflow is the JS-callable function.
// Usage: compileWorkflow(markdownString, filesObject?, filename?) → Promise<{yaml, warnings, error}>
//
// Arguments:
//   - markdownString: the main workflow markdown content
//   - filesObject (optional): a JS object mapping file paths to content strings,
//     used for import resolution (e.g. {"shared/tools.md": "---\ntools:..."})
//   - filename (optional): the source filename (e.g. "my-workflow.md"), defaults to "workflow.md"
func compileWorkflow(this js.Value, args []js.Value) any {
	if len(args) < 1 {
		return newRejectedPromise("compileWorkflow requires at least 1 argument: markdown string")
	}

	markdown := args[0].String()

	var files map[string][]byte
	if len(args) >= 2 && !args[1].IsNull() && !args[1].IsUndefined() {
		files = jsObjectToFileMap(args[1])
	}

	filename := "workflow.md"
	if len(args) >= 3 && !args[2].IsNull() && !args[2].IsUndefined() {
		filename = args[2].String()
	}

	var handler js.Func
	handler = js.FuncOf(func(this js.Value, promiseArgs []js.Value) any {
		resolve := promiseArgs[0]
		reject := promiseArgs[1]

		go func() {
			defer handler.Release()

			result, err := doCompile(markdown, files, filename)
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

// jsObjectToFileMap converts a JS object {path: content, ...} to map[string][]byte.
func jsObjectToFileMap(obj js.Value) map[string][]byte {
	files := make(map[string][]byte)
	keys := js.Global().Get("Object").Call("keys", obj)
	length := keys.Length()
	for i := 0; i < length; i++ {
		key := keys.Index(i).String()
		value := obj.Get(key).String()
		files[key] = []byte(value)
	}

	return files
}

// doCompile performs the actual compilation entirely in memory.
func doCompile(markdown string, files map[string][]byte, filename string) (js.Value, error) {
	// Set up virtual filesystem for import resolution
	if files != nil {
		parser.SetVirtualFiles(files)
		defer parser.ClearVirtualFiles()
	}

	// Derive workflow identifier from filename for fuzzy cron schedule scattering
	identifier := strings.TrimSuffix(filename, ".md")

	compiler := workflow.NewCompiler(
		workflow.WithNoEmit(true),
		workflow.WithSkipValidation(true),
		workflow.WithWorkflowIdentifier(identifier),
	)

	// Parse directly from string — no temp files needed
	workflowData, err := compiler.ParseWorkflowString(markdown, filename)
	if err != nil {
		return js.Undefined(), err
	}

	yamlContent, err := compiler.CompileToYAML(workflowData, filename)
	if err != nil {
		return js.Undefined(), err
	}

	result := js.Global().Get("Object").New()
	result.Set("yaml", yamlContent)
	result.Set("error", js.Null())
	result.Set("warnings", js.Global().Get("Array").New())
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

// getVersionFunc is the JS-callable function that returns the compiler version string.
// Usage: getVersion() → string
//
// The JavaScript runtime can call this to determine the current compiler version and
// compare it against the latest GitHub release to show an update notification to users.
func getVersionFunc(_ js.Value, _ []js.Value) any {
	return workflow.GetVersion()
}
