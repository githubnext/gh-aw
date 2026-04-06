// This file implements run-scripts validation for agentic workflows.
//
// # Run Scripts
//
// By default, the runtime manager adds --ignore-scripts to generated npm install
// commands to prevent pre/post install scripts from executing. This is a supply
// chain security measure: malicious packages can use install hooks to exfiltrate
// secrets or corrupt the runner environment.
//
// Users can opt in to install scripts by setting run-scripts: true in their
// workflow frontmatter. This emits a security warning in non-strict mode and
// is rejected as an error in strict mode.
//
// # Supported Flags (by package manager)
//
//   - npm / yarn / pnpm: --ignore-scripts
//   - pip / uv: no pre/post install lifecycle scripts (N/A)
//   - go / gem / bundle / dotnet / elixir / haskell / java / ruby: N/A
//
// # Configuration
//
// Global (all runtimes):
//
//	run-scripts: true
//
// Per-runtime (node only, since it is the only runtime that generates install commands):
//
//	runtimes:
//	  node:
//	    run-scripts: true

package workflow

import (
	"fmt"
	"os"

	"github.com/github/gh-aw/pkg/console"
)

var runScriptsLog = newValidationLogger("run_scripts")

// resolveRunScripts determines whether install scripts should be allowed based on
// the workflow frontmatter and any merged settings from imported shared workflows.
//
// Returns true (allow scripts) when any of the following is set:
//   - Global run-scripts: true in the top-level frontmatter
//   - runtimes.node.run-scripts: true in the frontmatter
//   - mergedRunScripts is true (any imported shared workflow enables run-scripts)
func resolveRunScripts(frontmatter map[string]any, runtimes map[string]any, mergedRunScripts bool) bool {
	// Already enabled by an imported shared workflow
	if mergedRunScripts {
		runScriptsLog.Print("run-scripts enabled by imported shared workflow")
		return true
	}

	// Check global run-scripts field
	if rsAny, ok := frontmatter["run-scripts"]; ok {
		if rsBool, ok := rsAny.(bool); ok && rsBool {
			runScriptsLog.Print("run-scripts enabled globally via run-scripts: true")
			return true
		}
	}

	// Check per-runtime run-scripts for node (the only runtime that generates npm install commands)
	if nodeAny, ok := runtimes["node"]; ok {
		if nodeMap, ok := nodeAny.(map[string]any); ok {
			if rsAny, ok := nodeMap["run-scripts"]; ok {
				if rsBool, ok := rsAny.(bool); ok && rsBool {
					runScriptsLog.Print("run-scripts enabled via runtimes.node.run-scripts: true")
					return true
				}
			}
		}
	}

	return false
}

// validateRunScripts emits a warning (non-strict mode) or returns an error (strict mode)
// when run-scripts is enabled in the workflow. This alerts users to the supply chain
// attack risk introduced by allowing npm pre/post install scripts.
func (c *Compiler) validateRunScripts(workflowData *WorkflowData) error {
	if !workflowData.RunScripts {
		runScriptsLog.Print("run-scripts not enabled, skipping validation")
		return nil
	}

	runScriptsLog.Print("run-scripts is enabled, emitting supply chain warning")

	warningMsg := "run-scripts: true is set – npm pre/post install scripts will execute during package installation. " +
		"This is a supply chain security risk: malicious or compromised packages can use install hooks to " +
		"exfiltrate secrets or tamper with the runner environment. " +
		"Remove run-scripts: true unless you fully trust all installed packages and their transitive dependencies."

	if c.strictMode {
		return fmt.Errorf("strict mode: %s", warningMsg)
	}

	fmt.Fprintln(os.Stderr, console.FormatWarningMessage(warningMsg))
	c.IncrementWarningCount()
	return nil
}
