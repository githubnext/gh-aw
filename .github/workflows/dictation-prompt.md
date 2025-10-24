---
name: Dictation Prompt Generator
on:
  workflow_dispatch:
  schedule:
    - cron: "0 6 * * 0"  # Weekly on Sundays at 6 AM UTC

permissions:
  contents: read
  actions: read

engine: copilot

imports:
  - shared/reporting.md

tools:
  edit:
  bash:
    - "*"
  github:
    allowed:
      - get_file_contents
      - search_code

safe-outputs:
  create-pull-request:
    title-prefix: "[docs] "
    labels: [documentation, automation]
    draft: false

timeout_minutes: 10
---

# Dictation Prompt Generator

**DO NOT PLAN. Just do the work immediately.**

Scan documentation files in `docs/src/content/docs/` and update `.github/instructions/dictation.instructions.md` with approximately 100 project-specific terms (95-105 acceptable). Focus on configuration keywords, engines, commands, GitHub concepts, file formats, and tool types. Exclude tooling-specific terms like makefile, Astro, and starlight. Alphabetize the glossary.

The file should contain:
- Title: Fix text-to-speech errors in dictated text
- Project Glossary: ~100 terms, alphabetically sorted, one per line
- Technical Context: Brief description of gh-aw
- Fix Speech-to-Text Errors: Common misrecognitions → correct terms

Create a pull request with title "[docs] Update dictation prompt instructions" when done.
