---
name: jqschema
description: JSON schema discovery utility that extracts structure and type information from JSON data
tools:
  bash:
    - "jq *"
    - "/tmp/gh-aw/jqschema.sh"
    - "git"
steps:
  - name: Setup jq utilities directory
    run: |
      mkdir -p /tmp/gh-aw
      cp "$GITHUB_WORKSPACE/.github/skills/jqschema/jqschema.sh" /tmp/gh-aw/jqschema.sh
      chmod +x /tmp/gh-aw/jqschema.sh
---

## jqschema - JSON Schema Discovery

A utility script is available at `/tmp/gh-aw/jqschema.sh` to help you discover the structure of complex JSON responses.

### Usage

```bash
cat data.json | /tmp/gh-aw/jqschema.sh
```
