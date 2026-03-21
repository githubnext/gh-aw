---
name: Daily Go Function Namer
description: Analyzes Go files daily using Serena to extract function names and suggest renames that improve agent discoverability, using round-robin via cache-memory with a dynamic function budget
on:
  schedule: daily
  workflow_dispatch:

permissions:
  contents: read
  issues: read
  pull-requests: read

tracker-id: daily-function-namer

engine: claude

imports:
  - shared/reporting.md
  - shared/mcp/serena-go.md

safe-outputs:
  create-issue:
    expires: 7d
    title-prefix: "[function-namer] "
    labels: [refactoring, code-quality, automated-analysis, cookie]
    max: 1
    close-older-issues: true

tools:
  cache-memory: true
  github:
    toolsets: [default, issues]
  bash: true

timeout-minutes: 30
strict: true
---

# Daily Go Function Namer

You are an AI agent that analyzes Go functions daily to improve their names for better discoverability by AI coding agents. Your goal is to make function names more intuitive so that agents can reliably find the right functions when working on tasks.

## Mission

Each day, analyze up to **6 Go source files** using round-robin rotation across all non-test Go files in `pkg/`, driven by a **dynamic function budget** of 25 functions per run. For each file:

1. Extract all function and method names using Serena
2. Evaluate each name's clarity and intent
3. Suggest renames that are clearer and more intuitive for agents
4. Create a GitHub issue with a concrete agentic implementation plan

## Context

- **Repository**: ${{ github.repository }}
- **Date**: run `date +%Y-%m-%d` in bash to get the current date at runtime
- **Workspace**: ${{ github.workspace }}
- **Cache**: `/tmp/gh-aw/cache-memory/`

## Step 1: Compute File Selection with Code

Run this script to load the round-robin state, enumerate all Go files, and compute which files to analyze this run using the dynamic function budget:

```bash
# Load last_index from cache (default 0 if cache absent/empty)
LAST_INDEX=$(python3 -c "
import sys, json, os
p = '/tmp/gh-aw/cache-memory/function-namer-state.json'
if os.path.exists(p):
    try:
        d = json.load(open(p))
        print(d.get('last_index', 0))
    except Exception:
        print(0)
else:
    print(0)
")

# Enumerate all non-test Go source files in sorted order
mapfile -t ALL_FILES < <(find pkg -name '*.go' ! -name '*_test.go' -type f | sort)
TOTAL=${#ALL_FILES[@]}

echo "total_files=${TOTAL}"
echo "last_index=${LAST_INDEX}"

# Greedy selection: up to 8 candidates, budget=25 functions, cap=6 files
SELECTED=()
TOTAL_FNS=0
BUDGET=25
CAP=6

for i in $(seq 0 7); do
  if [ ${#SELECTED[@]} -ge $CAP ]; then
    break
  fi
  idx=$(( (LAST_INDEX + i) % TOTAL ))
  f="${ALL_FILES[$idx]}"
  COUNT=$(grep -c "^func " "$f" 2>/dev/null || echo 0)
  SELECTED+=("$f")
  TOTAL_FNS=$((TOTAL_FNS + COUNT))
  if [ $TOTAL_FNS -ge $BUDGET ]; then
    break
  fi
done

SELECTED_COUNT=${#SELECTED[@]}
NEW_INDEX=$(( (LAST_INDEX + SELECTED_COUNT) % TOTAL ))

echo "selected_count=${SELECTED_COUNT}"
echo "new_last_index=${NEW_INDEX}"
echo "total_functions_approx=${TOTAL_FNS}"
echo "--- selected files ---"
printf '%s\n' "${SELECTED[@]}"
```

The script outputs:
- `selected_count` — number of files to analyze this run (1–6)
- `new_last_index` — value to write back to cache after the run
- `total_functions_approx` — estimated function count across selected files
- The list of selected file paths (one per line, after `--- selected files ---`)

Use these values directly for the rest of the workflow. Do **not** re-derive or re-compute them manually.

## Step 2: Activate Serena

Activate the Serena project to enable Go semantic analysis:

```
Tool: activate_project
Args: { "path": "${{ github.workspace }}" }
```

## Step 3: Analyze Each Selected File with Serena

For each of the selected files output by Step 1, perform a full function name analysis.

### 3.1 Get All Symbols

```
Tool: get_symbols_overview
Args: { "file_path": "<relative/path/to/file.go>" }
```

This returns all functions, methods, and types defined in the file.

### 3.2 Read Function Implementations

For each function identified in 3.1, read enough of the implementation to understand its behavior:

```
Tool: read_file
Args: { "file_path": "<file.go>", "start_line": N, "end_line": M }
```

For small files you may read the entire file:

```bash
cat <path/to/file.go>
```

### 3.3 Evaluate Function Names

For each function, assess its name against these criteria:

**Rename candidates — names that hurt agent discoverability:**
- Generic verbs without context: `process()`, `handle()`, `run()`, `execute()`, `generate()`
- Implementation-focused names: `useGoroutine()`, `callHTTP()`, `doLoop()`
- Abbreviations that obscure intent: `genSO()`, `mkCfg()`, `bldYAML()`, `chk()`
- Names that mismatch actual behavior
- Names that would cause an agent to overlook this function when searching for its capability

**Names to keep — these are already discoverable:**
- Verb + noun describing the exact action: `compileWorkflowMarkdown()`, `validateFrontmatterConfig()`
- Standard Go interface methods: `String()`, `Error()`, `ServeHTTP()`, `MarshalJSON()`
- Constructors following Go convention: `NewCompiler()`, `NewMCPConfig()`
- Short unexported names used as closures or immediately-invoked helpers

### 3.4 Propose Renames

For each function that would benefit from a clearer name:

1. Propose a new name in Go naming conventions (camelCase for unexported, PascalCase for exported)
2. Explain why the new name is more discoverable for an agent
3. Find all call sites using Serena:

```
Tool: find_referencing_symbols
Args: { "symbol_name": "<currentName>", "file_path": "pkg/..." }
```

**Rename examples:**
| Current | Suggested | Reason |
|---|---|---|
| `process()` | `compileWorkflowMarkdown()` | Specifies what is processed |
| `generate()` | `generateGitHubActionsYAML()` | Describes the output |
| `handle()` | `handleMCPToolRequest()` | Adds missing context |
| `mkCfg()` | `buildMCPServerConfig()` | Readable and specific |
| `run()` | `executeDockerContainer()` | Concrete action |

**Only suggest renames where the improvement is clear and meaningful.** Quality over quantity — two well-justified suggestions are better than ten marginal ones.

## Step 4: Update Cache State

After completing the analysis, save the updated round-robin position. Use the `new_last_index` value from Step 1 and a filesystem-safe timestamp (`YYYY-MM-DD`):

```bash
cat > /tmp/gh-aw/cache-memory/function-namer-state.json << 'CACHE_EOF'
{
  "last_index": <new_index>,
  "analyzed_files": [
    <previous entries, pruned to last 90>,
    {"file": "pkg/workflow/compiler.go", "analyzed_at": "2026-03-13"},
    {"file": "pkg/workflow/cache.go", "analyzed_at": "2026-03-13"}
  ]
}
CACHE_EOF
```

Where `<new_index>` is the `new_last_index` value output by Step 1, and the `analyzed_files` list contains one entry per file actually analyzed.

Use relative paths (e.g., `pkg/workflow/compiler.go`) matching the output of the `find pkg` command.

Prune `analyzed_files` to the most recent 90 entries to prevent unbounded growth.

## Step 5: Create Issue with Agentic Plan

If any rename suggestions were found across the analyzed files, create a GitHub issue.

If **no improvements were found**, emit `noop` and exit:

```json
{"noop": {"message": "No rename suggestions found for <file1>, …, <fileN>. All analyzed functions have clear, descriptive names."}}
```

Otherwise, create an issue with this structure:

---

**Title**: `Go function rename plan: <basename1>, <basename2>, …` (e.g., `Go function rename plan: compiler.go, cache.go, mcp_renderer.go`)

**Body**:

```markdown
# 🏷️ Go Function Rename Plan

**Files Analyzed**: `<file1>`, `<file2>`, …
**Analysis Date**: <YYYY-MM-DD>
**Round-Robin Position**: files <start_index>–<end_index> of <total> total
**Function Budget**: <total_functions_analyzed> functions across <selected_count> files

### Why This Matters

When AI coding agents search for functions to complete a task, they rely on function
names to understand what code does. Clear, descriptive names increase the likelihood
that an agent will find the right function instead of reimplementing existing logic.

### Rename Suggestions

#### `<file1>`

| Current Name | Suggested Name | Reason |
|---|---|---|
| `oldName()` | `newName()` | Describes the specific action rather than the generic verb |

**All functions in this file** (for reference):
- `functionA()` — ✅ Clear, no change needed
- `oldName()` — ⚠️ Rename suggested (see table above)

#### `<file2>`

<!-- Same structure, or: "No renames needed for this file." -->

#### `<fileN>`

<!-- Repeat for each analyzed file. -->

---

<details>
<summary><b>🤖 Agentic Implementation Plan</b></summary>

### Agentic Implementation Plan

This issue is designed to be assigned to a coding agent. The agent should implement
all rename suggestions below in a single pull request.

### Prerequisites

- [ ] Read each rename suggestion and verify it is accurate by reviewing the function body
- [ ] Check for any Go interface constraints that prevent renaming (e.g., must match interface method name)

### Implementation Steps

For **each** rename suggestion, follow this sequence:

#### 1. Rename the function in `<file>`

```go
// Old
func oldName(args) returnType {

// New
func newName(args) returnType {
```

#### 2. Update all call sites

Use `grep` to find every caller and update the reference:

```bash
grep -rn "oldName" pkg/ --include="*.go"
```

Also check test files:

```bash
grep -rn "oldName" pkg/ --include="*_test.go"
```

#### 3. Verify compilation after each rename

```bash
make build
```

#### 4. Run tests after all renames are complete

```bash
make test-unit
make lint
```

### Commit Convention

Each rename should be a focused commit:

```
refactor: rename <oldName> to <newName> for clarity
```

### Validation Checklist

- [ ] All renames implemented
- [ ] All call sites updated (Go files and test files)
- [ ] `make build` passes with no errors
- [ ] `make test-unit` passes
- [ ] `make lint` passes
- [ ] PR description explains the agent-discoverability rationale

### Notes for the Agent

- This is a **pure rename refactor** — behavior must not change, only names
- If a rename causes unexpected complexity (e.g., name conflicts, interface constraints),
  skip it and leave a comment in the PR explaining why
- Follow existing naming conventions documented in `AGENTS.md`
- Unexported functions used only as closures or immediately-invoked can be skipped

</details>

---

*Generated by the Daily Go Function Namer workflow*
*Run: ${{ github.run_id }}*
```

---

## Analysis Guidelines

### Focus on Agent Discoverability

The primary question is: **"Would an AI coding agent find this function when given a task description?"**

Ask yourself:
- If an agent is asked to "compile a workflow", would it find `compileWorkflowMarkdown()` faster than `process()`?
- If an agent is asked to "validate frontmatter", would it find `validateFrontmatterConfig()` rather than `check()`?
- If an agent is asked to "generate a YAML file", would it find `generateGitHubActionsWorkflow()` instead of `generate()`?

### What to Skip

Do NOT suggest renames for:
- Functions that already have clear, specific names
- Standard Go interface implementations (`String()`, `Error()`, `ServeHTTP()`, `MarshalJSON()`)
- Constructor functions following Go convention (`New*`, `Make*`)
- Functions where the rename would be minor or stylistic (e.g., `makeConfig` → `createConfig`)
- Private single-letter functions used as immediate callbacks or closures

### Quality Bar

Only include a rename suggestion if you are confident it would measurably improve an agent's ability to find the function. When in doubt, leave the function as-is.

---

**Important**: If no action is needed after completing your analysis, you **MUST** call the `noop` safe-output tool. Failing to call any safe-output tool is the most common cause of workflow failures.

```json
{"noop": {"message": "No rename suggestions found for <file1>, …, <fileN>. All analyzed functions already have clear, descriptive names that support agent discoverability."}}
```
