# ADR-52986: Inline Agentic Commands Router JavaScript

**Date**: 2026-08-15
**Status**: Draft
**Deciders**: pelikhan, copilot-swe-agent

---

### Context

The agentic commands router is a set of CommonJS JavaScript modules that implement slash-command routing logic for generated GitHub Actions workflows. Previously, these modules were distributed as external files loaded at runtime: the generated workflow included a `checkout` step plus a `gh-aw-actions/setup` step that copied the modules to `${{ runner.temp }}/gh-aw/actions/`, and the router entry point was then `require`d from that path. This approach added two extra workflow steps to every generated agentic command workflow, introduced latency at job startup, and created a version coupling risk between the Go compiler binary that generates workflows and the JavaScript modules fetched at runtime.

### Decision

We will embed all agentic commands JavaScript modules directly into the Go compiler binary using Go's `//go:embed` directive and generate a self-contained CommonJS bundle that is inlined into the `actions/github-script` step at workflow-generation time. The bundle uses a minimal module loader (`__ghAwRequire`) that resolves relative `require()` calls across the embedded module map without any filesystem access. The generated workflow no longer contains a `checkout` or `setup-action` step for the router.

### Alternatives Considered

#### Alternative 1: Keep External File Distribution (Checkout + Setup Action)

The existing model: a `gh-aw-actions/setup` action copies JavaScript modules to a well-known temp path, and the generated workflow `require`s the entry point from there. This is the simplest individual-module update path (update JS without a Go rebuild), but it adds two mandatory workflow steps, increases job startup time, and couples the runtime module version to a separate action release cycle rather than the compiler binary.

#### Alternative 2: Pre-Built External Bundle (esbuild/rollup artifact)

An external JavaScript bundler (esbuild, rollup, or webpack) could produce a single minified file committed to the repo, which the setup action copies rather than many individual modules. This eliminates step count parity concerns but still requires a runtime download step, adds a build-tool dependency to the release pipeline, and leaves the committed artifact out of sync with the Go compiler unless an automated check enforces it.

### Consequences

#### Positive
- Removes the `checkout` and `setup-action` steps from every generated agentic command workflow, reducing job startup time and step count.
- The JavaScript modules and the Go compiler binary share the same release artifact; there is no version skew between what generates the workflow and what runs inside it.
- Dependency resolution failures are caught at compile time (Go build error) rather than at workflow runtime.

#### Negative
- The Go compiler binary grows proportionally with the JavaScript module set; each new `.cjs` module added to the embed list increases binary size.
- JavaScript-only changes (bug fixes in a `.cjs` module) now require a full Go rebuild and release cycle; there is no way to patch the router without shipping a new binary.
- The generated workflow YAML file size increases substantially (~3,500 lines added to `agentic_commands.yml`) because the full module bundle is inlined; this makes the generated file harder to read directly.

#### Neutral
- The custom `__ghAwRequire` loader replicates a subset of Node.js `require` semantics (relative path resolution, module caching); it handles only the `.cjs`/`.js` extension set the agentic commands modules actually use.
- Tests for the bundler (`TestBundleAgenticCommandsScript`, `TestGetAgenticCommandsScript`) validate the Go-side bundling logic and embed correctness, but do not execute the inlined JavaScript end-to-end.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
