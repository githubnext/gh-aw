---
on: 
  workflow_dispatch:
concurrency:
  group: dev-workflow-${{ github.ref }}
  cancel-in-progress: true
name: Dev
engine: copilot
permissions:
  contents: read
  actions: read
tools:
  github:
imports:
  - shared/credsweeper.md
---

# Generate 3 Emoji Poem

Generate a creative 3-line poem using only emojis. Each line should tell part of a story or convey an emotion.

**Requirements:**
- Exactly 3 lines
- Only emojis (no text or words)
- Each line should have 3-5 emojis
- The poem should have a coherent theme or tell a simple story

**Examples of good emoji poems:**
```
🌅 ☕ 😊
🚶‍♂️ 🌳 🦋
🌙 ✨ 😴
```

After creating your poem, explain briefly what story or theme it represents.
