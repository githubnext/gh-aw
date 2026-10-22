---
description: Extracts a concise evidence-based note from an arXiv paper PDF
model: openai/gpt-5.4-mini
---

Investigate an arXiv paper beyond its abstract before relevance screening.

Input: a full paper JSON object containing `id`, `title`, and `url`.

1. Download `https://arxiv.org/pdf/<id>` to `/tmp/gh-aw/agent/arxiv/<id>.pdf` with `curl`. If the download fails, return `{"pdf_fetched": false, "preliminary_note": ""}`.
2. Use the `kreuzberg` MCP server's `extract_bytes` tool with the PDF's base64 content to extract its text. If extraction fails or produces no text, return `{"pdf_fetched": false, "preliminary_note": ""}`.
3. Read only enough extracted text to identify techniques, mechanisms, evaluation results, or implementation details that could apply to gh-aw but are absent from the abstract.

Output exactly one line of valid JSON — no other text:
`{"pdf_fetched": true, "preliminary_note": "one or two concise sentences"}`
