---
description: Fast relevance screening of arXiv paper abstracts for GitHub Agentic Workflows
model: openai/gpt-5.4-mini
---

Screen an arXiv paper abstract for relevance to GitHub Agentic Workflows (gh-aw).

gh-aw compiles markdown workflow files into GitHub Actions YAML, uses AI agents as workflow engines, manages tool access, enforces network firewalls, routes writes through typed safe-outputs, optimizes tokens, and supports multi-agent orchestration.

**Relevant** (any of):
- Agentic AI systems, multi-agent coordination, LLM agent workflows
- Prompt engineering, context management, or caching for LLM agents
- Token efficiency, structured output generation from LLMs
- AI-driven CI/CD, automated code review, workflow automation
- Security in agentic systems: sandboxing, tool access control, policy enforcement
- Orchestration patterns for decomposing tasks across AI agents

**Not relevant**:
- Pure mathematical theory with no LLM/agent application
- Hardware, systems, or networking research unrelated to AI
- Computer vision, speech, or domain-specific scientific tasks
- Medical, biological, or physical science applications

Input: `{"title": "...", "abstract": "...", "preliminary_note": "..."}` as a JSON string. Consider the preliminary note when present; do not reject a paper solely because the abstract is weak when the note identifies a relevant technique.

Output: exactly one line of valid JSON — no other text:
`{"relevant": true, "reason": "one sentence"}` or `{"relevant": false, "reason": "one sentence"}`
