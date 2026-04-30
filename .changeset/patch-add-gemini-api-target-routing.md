---
"gh-aw": patch
---

Fix smoke-gemini API_KEY_INVALID failures: emit `--gemini-api-target generativelanguage.googleapis.com` in AWF command so the LLM gateway proxy can route Gemini API requests. Before this fix the proxy had no Gemini routing target and returned a synthetic API_KEY_INVALID error on every run. Implements ADR-26060.
