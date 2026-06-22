---
name: jqschema
description: Infer JSON structure and types with jq-based schema discovery.
tools:
  bash:
    - "jq *"
    - "./.github/skills/jqschema/jqschema.sh"
    - "git"
---

## jqschema - JSON Schema Discovery

Use `./.github/skills/jqschema/jqschema.sh` to generate a compact structural schema (keys + types) from JSON input. Pipe any JSON source through it to discover structure before querying full data.

```bash
# Analyze a file or command output
cat data.json | ./.github/skills/jqschema/jqschema.sh
gh api search/repositories?q=language:go | ./.github/skills/jqschema/jqschema.sh
```

The script replaces object values with type names (`"string"`, `"number"`, `"boolean"`, `"null"`), reduces arrays to first-element structure, and outputs compact JSON. Use `perPage: 1` to fetch minimal data when exploring unknown API shapes.

**Example**: `{"total_count":1000,"items":[{"login":"user1","id":123}]}` → `{"total_count":"number","items":[{"login":"string","id":"number"}]}`
