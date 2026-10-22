---
title: Gemini CLI reference
description: Gemini CLI engine configuration, supported tools, runtime behavior, and limitations.
---

For first-time setup, see [Using Gemini CLI](/gh-aw/engines/gemini/). Shared options are in [AI engines](/gh-aw/reference/engines/).

## Selection and authentication

Use `engine: gemini` with `GEMINI_API_KEY` or [Google WIF](/gh-aw/reference/auth/#google-workload-identity-federation-wif). The default CLI pin is `0.64.0`; `engine.version` overrides it. Execution uses headless `stream-json` output and project-level `.gemini/settings.json` for tool restrictions and MCP configuration.

Safe-output threat detection falls back to Copilot, requiring separate Copilot authentication unless [another supported detection engine](/gh-aw/reference/threat-detection/) is selected.

The engine is deprecated but remains supported, with no removal date or automatic migration. [Agy migration constraints](/gh-aw/reference/engines/agy/#evaluating-a-gemini-api-key-workflow) apply before switching.

## Capabilities and limitations

Gemini supports top-level `max-turns`, custom API targets, and per-command bash allowlisting. Gemini does not provide native `tools.web-search`; configure an MCP search integration when needed. It also does not support bare mode, `max-continuations`, native `engine.agent` selection, or custom `engine.harness` scripts. See the [AI engine feature comparison](/gh-aw/reference/engines/#engine-feature-comparison).

Gemini does not resolve [model aliases](/gh-aw/reference/engines/#model-aliases) yet: the configured model reaches the CLI unchanged, and the compiler warns when `model:` is a gh-aw alias. Use an exact model id.

## Session artifacts

Gemini's native `agent-stdio.log` maps to the standard events in
`agent-session.jsonl`; conclusion publishes their essential fields in
`usage/aw_session.jsonl`. Messages retain exact text and streaming whitespace,
and tool events retain native IDs, structured arguments/results, and observed
outcomes. Understood provider errors use `session.error`, with non-warning
diagnostics also available in `session.result.errors`; system status uses
`session.info`, not an assistant answer. Identified message snapshots retain
their native history on core events without extra engine-specific wrapper events
or duplicate text. Existing native extension events remain supported.

Terminal accounting retains `sourceType: "result"` and reported zero values.
Gemini's `cached` tokens are already included in `input_tokens`; model subtotals
are not added again. Tool-call counts are not turns, and reasoning-token counts
are not inferred from a difference between total and input/output tokens.
Compatible reasoning and structured refusal observations use
`assistant.reasoning` and `assistant.refusal` when exposed; their absence from a
native stream does not imply an empty answer or a successful session.
See the [unified session specification](/gh-aw/specs/unified-agent-session-specification/#74-gemini)
for the mapping and existing-run fixture provenance.
