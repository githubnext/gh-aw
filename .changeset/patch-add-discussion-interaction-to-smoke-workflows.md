---
"gh-aw": patch
---

Add discussion interaction to smoke workflows; compiler now serializes the `discussion` flag into the safe-outputs handler config so workflows can post comments to discussions. Lock files include `discussions: write` where applicable.

Smoke workflows pick a random discussion and post a thematic comment (copilot: playful, claude: comic-book, codex: mystical oracle, opencode: space mission). This is a non-breaking tooling/workflow change.
