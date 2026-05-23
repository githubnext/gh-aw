---
name: jqschema
description: JSON schema discovery utility that extracts structure and type information from JSON data
tools:
  bash:
    - "jq *"
    - "./.github/skills/jqschema/jqschema.sh"
    - "git"
---

## jqschema - JSON Schema Discovery

A utility script is available directly in the repository skill folder at `./.github/skills/jqschema/jqschema.sh` to help you discover the structure of complex JSON responses.

### Usage

```bash
cat data.json | ./.github/skills/jqschema/jqschema.sh
```
