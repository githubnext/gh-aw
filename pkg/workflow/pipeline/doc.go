// Package pipeline implements the GitHub Actions pipeline intermediate
// representation, adapted from githubnext/ado-aw's src/compile/ir design.
//
// Build a Pipeline from typed jobs, steps, expressions, and output declarations.
// Resolve validates references and derives needs and job-output bindings without
// modifying the input. Lower converts the resolved graph to a YAML value tree;
// Emit serializes that tree once. RawStep is an escape hatch for authored YAML.
//
// Unlike Azure Pipelines, Actions has no stages, step IDs are scoped to a job,
// and cross-job step outputs must be explicitly exposed as job outputs. The
// resolver supplies those bindings automatically. It does not rewrite scripts:
// producers remain responsible for writing declared outputs to GITHUB_OUTPUT.
//
// The workflow package bridges its existing generators to this representation.
// Legacy YAML remains opaque and keeps its comments and formatting; newly
// migrated generators use typed steps and references.
//
// A producer declares OutputDecl{Name: "value"} on a RunStep with ID "produce".
// A consumer uses OutputRef{Job: "setup", Step: "produce", Name: "value"} in its
// Env, With, or Condition. Same-job references lower to steps.produce.outputs.value;
// cross-job references add a job-output binding and a needs edge automatically.
// Job-output references omit Step and must name an explicit Job.Outputs entry.
// Job conditions and job env cannot read steps from their own job; step
// references must point to an earlier step. Step IDs may repeat across jobs.
//
// RunStep.Script is a raw shell body, not YAML. Pass dynamic Actions values
// through Env; direct expression interpolation into typed run scripts is refused.
// Secret and Variable reference names, never their resolved values. Coalesce
// follows Actions' truthiness semantics, including false and zero.
//
// The legacy bridge accepts typed contributions through workflow.Job.Pipeline.
// Legacy steps run first, followed by typed steps. Keep envelope/native job
// properties on workflow.Job, and put typed conditions, env, output bindings, and
// steps in its Pipeline contribution. Do not set the same field in both forms.
// Resolution happens once after job augmentation and before final rendering;
// opaque legacy YAML is not reparsed or reserialized except to index step IDs
// in jobs with a typed contribution.
package pipeline
