---
description: Scores a relevant arXiv paper by actionability for GitHub Agentic Workflows
model: openai/gpt-5.4-mini
---

Score a relevant arXiv paper by how actionable it is for GitHub Agentic Workflows (gh-aw).

gh-aw compiles markdown workflows into GitHub Actions YAML with pluggable AI engines, safe-outputs typed writes, network firewall, token optimization, sub-agents, cache-memory, repo-memory, and multi-agent orchestration.

Score 1–5:
- 5: describes a new technique, pattern, or mechanism directly applicable to a specific gh-aw component
- 4: strong connection to gh-aw but requires adaptation
- 3: loosely related; possible indirect improvement
- 2: tangential; only marginally relevant to gh-aw
- 1: relevant to AI/agents generally but no clear gh-aw application

Input: `{"title": "...", "abstract": "...", "preliminary_note": "..."}` as a JSON string. Consider the preliminary note when present.

Output: exactly one line of valid JSON — no other text:
`{"score": <1-5>, "reason": "one sentence"}`
