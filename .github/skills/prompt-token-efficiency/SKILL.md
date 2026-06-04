---
name: prompt-token-efficiency
description: Rewrite prompts for minimal tokens, maximal clarity, and low ambiguity for LLM consumption.
---

# Prompt Token Efficiency

Use this skill to compress prompts while preserving intent and output quality.

## Goals

1. Minimize token count
2. Maximize clarity
3. Minimize ambiguity
4. Optimize for LLM execution, not human prose style

## Core Rules

- Keep only task-critical information.
- Remove pleasantries, repetition, and narrative framing.
- Prefer short, concrete instructions over descriptive paragraphs.
- Use explicit constraints and output format requirements.
- Use stable terminology (one term per concept).
- Replace vague words (`appropriate`, `some`, `better`) with measurable criteria.
- Put required context before optional context.
- Avoid conflicting instructions.

## Prose Compression Pattern

Rewrite prose to be direct and compact:

1. Start with the objective in one short sentence.
2. Keep only facts needed to complete the task.
3. Replace long qualifiers with concrete limits.
4. Remove filler words that do not change behavior.
5. End with explicit success criteria.

## LLM-Optimized Writing Style

- Use imperative statements.
- Prefer bullets over long prose.
- Keep each instruction atomic.
- Avoid examples unless needed to prevent failure.
- If examples are required, include one minimal example.

## Ambiguity Checks

Before finalizing a prompt, verify:

- Any undefined noun is resolved.
- Any pronoun has a clear antecedent.
- Scope limits are explicit (time range, file range, quantity limits).
- Success criteria are testable.
- Output format is unambiguous.

## Prose Rewrite Checks

When rewriting, ensure the final prompt:

- Uses fewer words than the original.
- Preserves all required constraints.
- Uses concrete nouns instead of pronouns where possible.
- Avoids optional wording unless options are actually allowed.
- States required output and any length bound in plain language.
