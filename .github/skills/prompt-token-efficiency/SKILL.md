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

## Compression Pattern

Rewrite prompts in this order:

1. **Task**: one sentence objective.
2. **Inputs**: exact data and boundaries.
3. **Constraints**: hard requirements and forbidden behavior.
4. **Output**: strict schema/shape and brevity target.

## LLM-Optimized Writing Style

- Use imperative statements.
- Prefer bullets over long prose.
- Keep each instruction atomic.
- Use compact field labels (`Task`, `Input`, `Constraints`, `Output`).
- Avoid examples unless needed to prevent failure.
- If examples are required, include one minimal example.

## Ambiguity Checks

Before finalizing a prompt, verify:

- Any undefined noun is resolved.
- Any pronoun has a clear antecedent.
- Scope limits are explicit (time range, file range, quantity limits).
- Success criteria are testable.
- Output format is unambiguous.

## Output Contract Template

Use this compact shape when rewriting prompts:

```text
Task: <single objective>
Input: <required context only>
Constraints:
- <hard rule 1>
- <hard rule 2>
Output:
- Format: <exact structure>
- Length: <token/line/word bound>
- Include: <required fields>
- Exclude: <forbidden content>
```

