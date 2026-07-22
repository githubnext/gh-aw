---
# Shared AI coding terminology reference for workflow prompts.
#
# Source: https://github.com/mattpocock/dictionary-of-ai-coding
# Purpose: Keep AI-coding terminology consistent across workflow outputs.
---

# AI Coding Dictionary Reference

Use the AI Coding Dictionary as a secondary terminology authority:

- Source: https://github.com/mattpocock/dictionary-of-ai-coding
- Website: https://aicodingdictionary.com

## Reusable Summary

The dictionary organizes terminology across these areas:

1. **The model** (model, tokens, inference, provider requests, cache tokens)
2. **Sessions and context** (context window, system prompt, turn, stateless/stateful)
3. **Tools and environment** (tool call/result, MCP, sandbox, permission modes)
4. **Failure modes** (hallucination, attention degradation, knowledge cutoff)
5. **Handoffs** (primary/secondary sources, handoff artifacts, compaction)
6. **Memory and steering** (memory systems, AGENTS.md, progressive disclosure, skills, subagents)
7. **Patterns of work** (human-in-the-loop, review patterns, prototyping, DX/AX)

## Usage Rules

- Prefer repository-specific terminology from `docs/src/content/docs/reference/glossary.md` when there is a conflict.
- Use this dictionary to disambiguate generic AI terms and reduce wording drift across docs and workflow output.
- When introducing or revising AI-coding terms, include a short "AI dictionary alignment summary" in your notes/PR description with:
  - terms reviewed
  - terms adopted
  - conflicts resolved (if any)
