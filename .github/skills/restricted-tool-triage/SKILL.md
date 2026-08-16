---
name: restricted-tool-triage
description: Operate safely and efficiently inside a gh-aw workflow with a restricted tools/bash allowlist, and correctly triage tool-denial events before they exhaust the session's denial budget.
---

# Restricted Tool Triage

Use this skill whenever you (the agent) are executing inside a gh-aw workflow whose frontmatter declares a narrow `tools:` allowlist (e.g. a short `bash: [...]` list, a scoped MCP `toolsets`, or no `read`/`shell` at all) and you hit — or are at risk of hitting — a "permission denied" / tool-denial response from the harness.

## Why this matters

gh-aw enforces a **hard, non-renewable denial budget** per session (commonly 3 denied tool calls). Once the threshold is reached, the harness emits `guard.tool_denials_exceeded` and aborts the entire session immediately — no further turns, no partial credit, no chance to recover. Treat every tool denial as spending down a scarce budget, not as a way to probe what's allowed.

## Triggers

- A tool call returns "permission denied by workflow tool permissions" or similar.
- You are about to try a shell/read/write command and are unsure if it's in the declared `tools:` allowlist.
- The workflow frontmatter shows a short/explicit `bash:` list, restrictive MCP `toolsets`, or omits `edit`/`bash` entirely.

## Procedure

1. **Read the allowlist first, before acting.** Before issuing any shell/file/MCP command, check the workflow's declared `tools:` block (frontmatter `bash: [...]`, `edit:`, MCP `toolsets:`, etc.) if visible in context, or infer it from the first denial message, which echoes the exact denied command. Do not assume general-purpose shell access is available just because the environment looks like a normal shell.

2. **On the first denial, stop and pivot — do not retry variants.** A denial is not a request to try a slightly different phrasing of the same disallowed command (e.g. don't go from `git status` to `git status --short` to `git diff --stat` as three separate attempts). Instead:
   - Identify the *capability* you actually need (e.g. "see which files changed").
   - Map it to a tool/command explicitly present in the allowlist (e.g. use `git diff --name-only` if `git diff:*` is allowed but `git status` is not; use the already-available MCP toolset instead of raw `read`/`shell` for file or repo introspection).
   - If no allowed tool can achieve the capability, stop attempting workarounds for that capability and route around it (skip the sub-task, or note the limitation in your output) rather than spending more of the denial budget.

3. **Budget awareness.** Assume a low, fixed denial ceiling (verify from harness messages such as "N/M" if shown, e.g. "tool denial 2/3"). Once you're at 1 remaining denial, do not attempt anything speculative — only proceed with actions you are confident are allowed.

4. **Don't misreport scope-as-bug.** A restricted toolset is very often an intentional, security-motivated author choice (least-privilege workflow design), not a misconfiguration. Before calling `missing_tool` / `missing_data` / equivalent "report a gap" safe-output:
   - Confirm the missing capability is genuinely required to complete the task and has no in-allowlist substitute.
   - Do NOT claim "verify token scopes / repository permissions / credentials" when the actual evidence is a `tools:` allowlist denial — that phrasing wrongly suggests an infra/auth bug and can prompt maintainers to loosen permissions unnecessarily, which is a security regression.
   - If you do report a gap, name the specific missing tool/capability and cite the exact denied command(s), not a generic "permissions" narrative.

5. **Prefer completing partial work over aborting.** If some parts of the task can be completed using only allowed tools, finish and report those, and clearly note what could not be done due to the restricted toolset — rather than continuing to probe disallowed tools until the session is forcibly terminated.

## Verification checklist

- [ ] Did you check the declared `tools:` allowlist (or infer it from the first denial) before issuing further commands?
- [ ] After any denial, did you switch to a different *capability strategy* rather than retry a denied command class?
- [ ] Did you stay well under the denial threshold (ideally 0 denials, never risk the last one on a speculative call)?
- [ ] If you reported a missing tool/capability, did you cite the specific denied command(s) instead of a generic "check credentials/permissions" claim?

## Stop conditions

- If you reach 2 denials, stop attempting anything not certain to be in the allowlist — finish with only allowed tools and report the limitation.
- If the required capability has no allowed substitute, do not keep probing; complete what you can and clearly state the constraint in your final output rather than exhausting the denial budget.
