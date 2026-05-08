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
      cat > /tmp/gh-aw/jqschema.sh << 'EOF'
      #!/usr/bin/env bash
      # jqschema.sh
      jq -c '
      def walk(f):
        . as $in |
        if type == "object" then
          reduce keys[] as $k ({}; . + {($k): ($in[$k] | walk(f))})
        elif type == "array" then
          if length == 0 then [] else [.[0] | walk(f)] end
        else
          type
        end;
      walk(.)
      '
      EOF
      chmod +x /tmp/gh-aw/jqschema.sh
---

## jqschema - JSON Schema Discovery

A utility script is available at `/tmp/gh-aw/jqschema.sh` to help you discover the structure of complex JSON responses.

### Usage

```bash
cat data.json | /tmp/gh-aw/jqschema.sh
```
