---
title: YAML Compatibility
description: Understanding YAML 1.2 parser requirements and compatibility issues with external tooling.
sidebar:
  order: 400
---

This reference documents the YAML parser requirements for GitHub Agentic Workflows and addresses compatibility issues with external validation tooling.

## YAML 1.2 Requirement

GitHub Agentic Workflows uses [goccy/go-yaml](https://github.com/goccy/go-yaml), a YAML 1.2 compliant parser. This is a critical requirement due to how the `on` field is handled in different YAML versions.

### The `on` Field Issue

The `on` field is **required** in all agentic workflows to define trigger events:

```yaml
---
on:
  schedule:
    - cron: "0 9 * * *"
---
```

However, `on` is a **reserved boolean keyword** in YAML 1.1, along with `yes`, `no`, and `off`. YAML 1.1 parsers interpret these as boolean values:

- `on:` → `true`
- `off:` → `false`
- `yes:` → `true`
- `no:` → `false`

This creates a fundamental incompatibility with external tooling that uses YAML 1.1 parsers.

## Parser Behavior Comparison

### YAML 1.2 Parser (Correct)

YAML 1.2 removed the implicit boolean keywords, treating `on` as a regular string key:

```python
# Using ruamel.yaml (YAML 1.2)
from ruamel.yaml import YAML

yaml_text = """
on:
  schedule:
    - cron: "0 9 * * *"
"""

yaml = YAML()
data = yaml.load(yaml_text)
print(data)
# Output: {'on': {'schedule': [{'cron': '0 9 * * *'}]}}
```

### YAML 1.1 Parser (Incorrect)

YAML 1.1 parsers like PyYAML interpret `on:` as a boolean:

```python
# Using PyYAML (YAML 1.1)
import yaml

yaml_text = """
on:
  schedule:
    - cron: "0 9 * * *"
"""

data = yaml.safe_load(yaml_text)
print(data)
# Output: {True: {'schedule': [{'cron': '0 9 * * *'}]}}
# ^^^^ Boolean key instead of string key!
```

This breaks JSON schema validation and any tooling expecting `on` to be a string key.

## Impact on External Tooling

### JSON Schema Validation

External tools using YAML 1.1 parsers will fail schema validation:

```bash
# Using a YAML 1.1 parser with JSON schema validation
$ pykwalify -s workflow.schema.json -d my-workflow.md
ERROR: Key 'on' not found. Got keys: [True]
```

The schema expects a string key named `"on"`, but the YAML 1.1 parser provides a boolean key `true`.

### IDE Integration

IDEs and editors using YAML 1.1 parsers for validation will show false errors:

- ❌ "Required property 'on' is missing"
- ❌ "Additional property 'true' is not allowed"
- ❌ Schema validation fails even though the workflow is valid

## Solutions for External Tools

### Option 1: Use YAML 1.2 Parsers

Replace YAML 1.1 parsers with YAML 1.2 compliant alternatives:

**Python:**
```bash
# Replace PyYAML with ruamel.yaml
pip install ruamel.yaml
```

```python
from ruamel.yaml import YAML

yaml = YAML()
data = yaml.load(workflow_file)
```

**JavaScript/TypeScript:**
```bash
# Use yaml package (YAML 1.2 compliant)
npm install yaml
```

```javascript
import { parse } from 'yaml'

const data = parse(workflowContent)
```

**Go:**
```bash
# Use goccy/go-yaml (same as gh-aw)
go get github.com/goccy/go-yaml
```

```go
import "github.com/goccy/go-yaml"

var data map[string]any
yaml.Unmarshal(content, &data)
```

### Option 2: Quote the `on` Key (Workaround)

Some YAML 1.1 parsers correctly handle quoted keys:

```yaml
---
"on":
  schedule:
    - cron: "0 9 * * *"
---
```

**⚠️ Warning:** This is not officially supported by gh-aw and may cause issues. The canonical solution is to use YAML 1.2 parsers.

### Option 3: Pre-process YAML

Transform the YAML before validation:

```python
import re
import yaml

# Read the frontmatter
with open('workflow.md', 'r') as f:
    content = f.read()

# Extract frontmatter
match = re.match(r'^---\n(.*?)\n---', content, re.DOTALL)
if match:
    frontmatter = match.group(1)
    # Replace 'on:' with quoted version
    frontmatter = re.sub(r'\bon:', '"on":', frontmatter)
    # Now parse with PyYAML
    data = yaml.safe_load(frontmatter)
```

## Validating Workflows

### Using gh-aw (Recommended)

The official `gh aw` CLI handles YAML parsing correctly:

```bash
# Compile and validate workflow
gh aw compile my-workflow.md

# View validation errors
gh aw compile --verbose
```

### Schema-Only Validation

If you need to validate against the JSON schema without the CLI:

1. **Extract frontmatter** from the markdown file
2. **Parse with YAML 1.2** parser (ruamel.yaml, yaml npm package, goccy/go-yaml)
3. **Validate against schema** using a JSON schema validator

Example in Python:

```python
from ruamel.yaml import YAML
import jsonschema
import json

# Load schema
with open('workflow.schema.json', 'r') as f:
    schema = json.load(f)

# Extract and parse frontmatter
yaml = YAML()
with open('workflow.md', 'r') as f:
    content = f.read()
    # Extract frontmatter between --- markers
    frontmatter = content.split('---')[1]
    data = yaml.load(frontmatter)

# Validate
jsonschema.validate(instance=data, schema=schema)
print("✓ Workflow is valid")
```

## Why GitHub Actions Works

GitHub Actions itself uses YAML 1.2 compliant parsers, so the `on` field works correctly in generated `.lock.yml` files. The issue only affects external validation tooling using YAML 1.1 parsers.

## Summary

- **gh-aw requires YAML 1.2** due to the `on` field requirement
- **YAML 1.1 parsers fail** by treating `on:` as boolean `true`
- **External tools must use YAML 1.2 parsers** for correct validation
- **Recommended parsers:** ruamel.yaml (Python), yaml (JavaScript), goccy/go-yaml (Go)
- **Use `gh aw compile`** for official validation

## Additional Resources

- [YAML 1.2 Specification](https://yaml.org/spec/1.2/spec.html)
- [YAML 1.1 to 1.2 Changes](https://yaml.org/spec/1.2/spec.html#id2761803)
- [goccy/go-yaml Documentation](https://github.com/goccy/go-yaml)
- [ruamel.yaml Documentation](https://yaml.readthedocs.io/)
