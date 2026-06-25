---
private: true
emoji: "🔒"
name: Sandbox Sudo False Rollout
description: Enable network-isolation mode (sandbox.agent.sudo false) for 25% of agentic workflows that currently have sudo true. Creates a pull request with all frontmatter edits and recompiled lock files.
on:
  workflow_dispatch:
permissions:
  contents: read
  pull-requests: read
engine:
  id: copilot
  copilot-sdk: true
strict: true
network:
  allowed:
    - defaults
tools:
  cli-proxy: true
  edit:
  bash:
    - "grep *"
    - "find *"
    - "sort *"
    - "head *"
    - "wc *"
    - "echo *"
    - "cat *"
    - "python3 *"
    - "basename *"
    - "gh aw compile *"
safe-outputs:
  create-pull-request:
    title-prefix: "[sandbox-rollout] "
    labels: [security, sandbox, automation]
    draft: true
    expires: 14d
    allow-workflows: true
    allowed-files:
      - .github/workflows/*.md
      - .github/workflows/*.lock.yml
  noop:
imports:
  - shared/app-config.md
  - shared/otlp.md
timeout-minutes: 30
sandbox:
  agent:
    sudo: true
---

# Sandbox Sudo False Rollout

You are a security hardening agent. Your mission: enable **network-isolation mode** (`sandbox.agent.sudo: false`) for exactly **25% of the agentic workflows** in this repository that currently have `sandbox.agent.sudo: true` set in their YAML frontmatter.

When `sudo: false` is set, the AWF runtime runs rootless (no sudo) and in network-isolation mode — a stricter security posture that limits privilege escalation inside the agent sandbox.

## Step 1 — Discover eligible workflows

Run the following Python script to find all eligible workflow files. It parses only the YAML frontmatter (between the first two `---` markers) so code examples in the prompt body cannot cause false positives:

```bash
python3 - <<'EOF'
import os, re, sys

workflows_dir = ".github/workflows"
eligible = []

for fname in sorted(os.listdir(workflows_dir)):
    if not fname.endswith(".md"):
        continue
    path = os.path.join(workflows_dir, fname)
    try:
        with open(path) as f:
            content = f.read()
    except Exception:
        continue
    # Extract YAML frontmatter (between first two --- markers)
    match = re.match(r"^---\n(.*?)\n---", content, re.DOTALL)
    if not match:
        continue
    frontmatter = match.group(1)
    # Check sandbox.agent.sudo: true — look for the indented YAML key
    if re.search(r"^\s+sudo:\s+true\s*$", frontmatter, re.MULTILINE):
        eligible.append(path)

total = len(eligible)
target = (total + 3) // 4  # ceil(total * 0.25)
print(f"TOTAL={total}")
print(f"TARGET={target}")
for p in eligible:
    print(f"FILE={p}")
EOF
```

Parse the output:
- `TOTAL` = total number of eligible workflows
- `TARGET` = number to convert (ceil of 25%)
- `FILE=...` lines = the sorted eligible workflow paths

Take the **first `TARGET` files** from the `FILE=` list (already sorted alphabetically).

If `TARGET` is 0 or no eligible workflows were found, call `noop` and stop.

## Step 2 — Apply the change to each selected workflow

For each of the `TARGET` selected workflow files:

1. **Read the file** to confirm the exact line containing `sudo: true` within the frontmatter sandbox block.

2. **Edit the file** using the `edit` tool: replace the `sudo: true` line with `sudo: false`.
   - Replace only the first occurrence of `    sudo: true` (4-space indent) that appears in the frontmatter (before the second `---` separator).
   - Do not modify any `sudo: true` text appearing in the prompt body (after the second `---`).

3. **Recompile the lock file**:
   ```bash
   gh aw compile .github/workflows/<workflow-basename-without-.md>
   ```
   Run this after editing each `.md` file so the corresponding `.lock.yml` is regenerated to reflect the rootless AWF mode. If the compile command fails, note the error but continue with the remaining workflows.

## Step 3 — Create the pull request

After processing all selected workflows, call `create-pull-request` with:

- **Title**: `Enable sandbox network-isolation mode for <TARGET> workflows (<TOTAL> total eligible)`
- **Body** (use `###` headers):

```
### Summary

Enable `sandbox.agent.sudo: false` (network-isolation / rootless mode) for <TARGET> of the <TOTAL> agentic workflows that currently have `sudo: true`.

This is a **25% incremental rollout**. When `sudo: false` is set, the AWF runner uses `--rootless` mode — removing the sudo privilege from the agent sandbox and enabling stricter network isolation.

### Changed workflows

<bullet list of the modified workflow basenames>

### Security impact

- AWF runs without sudo inside the sandbox
- Stricter privilege boundary for the agent runtime
- No functional change to workflow outputs or GitHub API access

### Next steps

After this PR is merged and verified, subsequent rollout PRs can cover the remaining eligible workflows.
```

If no changes were made (all compile steps failed or no eligible files were found), call `noop` instead.
