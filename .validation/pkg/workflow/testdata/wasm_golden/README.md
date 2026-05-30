# Wasm Golden Tests

Golden file tests that verify the wasm compiler (string API) produces correct YAML output.

## Directory structure

```
wasm_golden/
  fixtures/                          # Input .md workflow files
    basic-copilot.md                 # Synthetic fixtures (stable)
    smoke-claude.md                  # Smoke workflow fixtures (from .github/workflows/)
    shared/                          # Shared components for import resolution
      mood.md
      reporting.md
      ...
  TestWasmGolden_CompileFixtures/    # Golden output files (auto-generated)
    basic-copilot.golden
    smoke-claude.golden
    ...
```

## How it works

1. Each `.md` file in `fixtures/` is compiled via `ParseWorkflowString()` + `CompileToYAML()` — the same code path used by the wasm binary
2. The output is compared byte-for-byte against the corresponding `.golden` file
3. The Node.js wasm test (`scripts/test-wasm-golden.mjs`) also builds the actual wasm binary and verifies its output matches the same golden files

## Common tasks

### Regenerate golden files after compiler changes

If you change the compiler and golden tests fail, regenerate the expected output:

```bash
make update-wasm-golden
```

This runs `go test ./pkg/workflow -run='^TestWasmGolden_' -update` which overwrites the `.golden` files with current compiler output. Review the diff before committing.

### Add a new fixture

1. Create a `.md` file in `fixtures/` with valid frontmatter (`name`, `on`, `engine`)
2. If it uses `imports:`, add the shared components to `fixtures/shared/`
3. Generate the golden file:
   ```bash
   make update-wasm-golden
   ```
4. Commit the new `.md` file and its `.golden` file together

### Run just the wasm golden tests

```bash
# Go string API tests (fast, ~0.5s)
make test-wasm-golden

# Full wasm binary test via Node.js (builds wasm first, ~30s)
make test-wasm
```

### Fix a failing golden test

Golden tests fail when the compiler output changes. This is expected after compiler changes — the test is doing its job.

1. Run the failing test to see the diff:
   ```bash
   go test -v -timeout=5m -run='^TestWasmGolden_CompileFixtures/basic-copilot$' ./pkg/workflow
   ```
2. If the change is intentional, regenerate:
   ```bash
   make update-wasm-golden
   ```
3. Review the golden file diff (`git diff`) to confirm the changes are expected
4. Commit the updated `.golden` files with your compiler change

### Fix a wasm-specific divergence

If the Node.js wasm test fails but the Go golden test passes, the wasm binary is producing different output than the native string API. Common causes:

- **File reading**: Code using `os.ReadFile` instead of `parser.ReadFile` — the wasm build overrides `parser.ReadFile` to check virtual files
- **contentOverride**: The `Compiler.contentOverride` field must remain set until `CompileToYAML` completes (cleared in its defer, not in `ParseWorkflowString`)
- **Missing build tag stubs**: New code that calls OS functions needs a wasm stub (see `*_wasm.go` files in `pkg/workflow/` and `pkg/parser/`)

## Design decisions

**Why not use production workflows as fixtures?**
Production workflows in `.github/workflows/` change frequently. Using them as fixtures would break the golden tests on every mundane workflow edit, creating friction. The fixtures are self-contained copies that only change when the compiler changes.

**Why smoke workflows?**
The 4 smoke workflows (claude, codex, copilot, test-tools) provide real-world coverage with imports, safe-outputs, tools, network config, and MCP servers — features the synthetic fixtures don't exercise.

**Why exact match (no tolerance)?**
The golden test verifies compiler determinism. Any output difference — even a single character — indicates a real change in compilation behavior that should be reviewed.
