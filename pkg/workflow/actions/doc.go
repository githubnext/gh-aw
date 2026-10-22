// Package actions defines the GitHub Actions workflow AST.
//
// Workflow models native on, permissions, env, defaults, concurrency, and jobs
// properties. A Job either runs steps on RunsOn or calls a reusable workflow
// through Uses, With, and Secrets. Steps use Run or Uses and native If and Env
// properties. There are no platform-neutral envelopes or alternate CI targets.
//
// RunsOn retains native scalar, label-sequence, and group/labels mapping values.
// Authored runner expressions remain strings and are never evaluated here.
//
// Resolve validates output references and derives needs and job-output bindings
// without modifying the input. Lower builds a deterministic YAML value tree;
// Emit serializes it once. RawStep accepts authored GitHub Actions step mappings
// and rejects non-Actions properties.
//
// Parse reads native workflow YAML and validates its dependency graph. Authored
// expression strings remain opaque; explicit needs are authoritative. AW
// Markdown source documents are modeled separately by the aw package.
// Parse followed by Emit round-trips the original YAML byte-for-byte, including
// comments, field order, quoting, and line endings. Resolve retains this source.
// After edits, Emit reconciles changes with the original syntax, preserving
// comments on retained nodes and moving step comments with their steps.
// Edited regions may be reformatted; aliases are materialized to preserve
// independent AST values. Lower remains a semantic tree without source comments.
//
// Producers declare OutputDecl on a named step and write values to GITHUB_OUTPUT.
// Consumers use OutputRef in If, Env, With, or Secrets. Same-job references become
// steps.<id>.outputs.<name>; cross-job references expose a job output and add a
// needs edge. References with an empty Step name an explicit Job.Outputs entry
// or an output declared in a reusable-workflow caller's CallOutputs.
//
// Step IDs are job-scoped. A step can only read outputs of earlier steps in its
// job, and job-level properties cannot read their own step outputs. Workflow env
// cannot reference job or step outputs.
//
// Run contains a shell body, not YAML. Dynamic Actions values must pass through
// Env instead of being interpolated into scripts. Secret and Variable reference
// names, never resolved values. Or uses the Actions || truthiness rules.
//
// Existing compiler generators contribute typed fields through workflow.Job.Actions.
// Existing steps run before typed steps, and their YAML comments and formatting
// are retained. The compiler resolves contributions after job augmentation and
// before rendering. Do not set a field in both its existing and typed form.
package actions
