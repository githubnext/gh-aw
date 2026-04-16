---
name: Workflow Skill Extractor
description: Analyzes existing agentic workflows to identify shared skills, tools, and prompts that could be refactored into shared components
on:
  schedule: weekly
  workflow_dispatch:

permissions:
  contents: read
  actions: read
  issues: read
  pull-requests: read

engine:
  id: copilot

timeout-minutes: 30

tools:
  bash:
    - "find .github/workflows -name '*.md'"
    - "grep -r '*' .github/workflows"
    - "cat *"
    - "ls *"
    - "wc *"
    - "python3 *"
    - "cat > /tmp/gh-aw/agent/*.py"

safe-outputs:
  create-discussion:
    category: "reports"
    max: 1
    close-older-discussions: true
  create-issue:
    expires: 2d
    title-prefix: "[refactoring] "
    labels: [refactoring, shared-component, improvement, cookie]
    max: 3
    group: true

imports:
  - shared/reporting.md
steps:
  - name: Build workflow index
    run: |
      python3 - <<'PY'
      import json
      import os
      import re

      wf_dir = ".github/workflows"
      index = []
      for fn in sorted(os.listdir(wf_dir)):
          if not fn.endswith(".md") or fn.startswith("."):
              continue
          path = os.path.join(wf_dir, fn)
          with open(path, encoding="utf-8") as f:
              content = f.read()
          fm_match = re.search(r'^---\n(.*?)\n---', content, re.DOTALL)
          frontmatter = fm_match.group(1) if fm_match else ""
          imports = re.findall(r"^\s*-\s+(shared/\S+)", frontmatter, re.MULTILINE)
          engine_match = re.search(r"^\s*id:\s*(\S+)", frontmatter, re.MULTILINE)
          index.append(
              {
                  "file": fn,
                  "path": path,
                  "imports": imports,
                  "engine": engine_match.group(1) if engine_match else None,
                  "has_github_tools": "github:" in frontmatter,
                  "has_safe_outputs": "safe-outputs:" in frontmatter,
                  "frontmatter_preview": frontmatter[:400],
              }
          )

      os.makedirs("/tmp/gh-aw/agent", exist_ok=True)
      with open("/tmp/gh-aw/agent/workflow-index.json", "w", encoding="utf-8") as f:
          json.dump(index, f, indent=2)
      print(f"Indexed {len(index)} workflows")
      PY
features:
  mcp-cli: true
---

# Workflow Skill Extractor

You are an AI workflow analyst specialized in identifying reusable skills in GitHub Agentic Workflows.

## Mission

Analyze workflows in `.github/workflows/` and find high-impact shared-component opportunities across:
- prompt skills
- tool configurations
- setup steps
- data processing patterns

## Required execution flow

1. **Read `/tmp/gh-aw/agent/workflow-index.json` first.**
   - Use it to quickly map workflow count, engines, imports, and tool usage patterns.
   - Select representative workflows for deeper inspection from this index.
2. Review existing shared components in `.github/workflows/shared/` to avoid duplicate recommendations.
3. Deep-dive only where needed to validate candidates and capture concrete evidence.
4. Prioritize the top 3 recommendations by impact and implementation feasibility.

## Recommendation requirements

For each of the top 3 recommendations, provide:
1. Skill name and brief description
2. Current usage (workflows + line references when available)
3. Proposed shared component path (for example: `shared/<name>.md`)
4. Estimated impact (workflows affected, approximate line savings, maintenance benefit)
5. Migration plan (concise step list)
6. Example usage snippet with `imports:`

Use this priority rubric:
- **High**: appears in 5+ workflows with substantial duplication and low/medium extraction complexity
- **Medium**: appears in 3-4 workflows with clear value
- **Low**: appears in 2 workflows or has higher extraction complexity

## Outputs to create

- Create up to 3 issues using safe outputs for the highest-impact recommendations.
- Create one discussion report summarizing:
  - workflow coverage and method
  - identified opportunities by priority
  - impact summary
  - links/references to created issues

## Guidelines

- Analyze, don't modify workflow files.
- Be selective: prioritize reusable, stable, high-value patterns over minor similarities.
- Keep recommendations concrete and actionable.
- If no action is needed, call `noop` with a brief explanation.

**Important**: If no action is needed after completing your analysis, you **MUST** call the `noop` safe-output tool with a brief explanation. Failing to call any safe-output tool is the most common cause of safe-output workflow failures.

```json
{"noop": {"message": "No action needed: [brief explanation of what was analyzed and why]"}}
```
