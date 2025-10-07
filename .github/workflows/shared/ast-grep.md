---
tools:
  bash: ["ast-grep:*"]
steps:
  - name: Install ast-grep
    run: |
      npm install --global @ast-grep/cli
      ast-grep --version
---

## ast-grep Tool Setup

### Using ast-grep

ast-grep is a powerful structural search and replace tool for code. It uses tree-sitter grammars to parse and search code based on its structure rather than just text patterns.

### Basic Usage

**Search for patterns:**
```bash
ast-grep --pattern '$PATTERN' --lang go
```

**Search in specific files:**
```bash
ast-grep --pattern '$PATTERN' --lang go path/to/files/**/*.go
```

**Common Go patterns to detect:**

1. **Unmarshal with dash tag** (problematic pattern):
   ```bash
   ast-grep --pattern 'json:"-"' --lang go
   ```

2. **Error handling issues:**
   ```bash
   ast-grep --pattern 'if err != nil { $$$A }' --lang go
   ```

3. **Finding specific function calls:**
   ```bash
   ast-grep --pattern 'functionName($$$ARGS)' --lang go
   ```

### Output Format

By default, ast-grep outputs matched code with line numbers and context. Use `--json` flag for machine-readable output:
```bash
ast-grep --pattern '$PATTERN' --lang go --json
```

### More Information

- Documentation: https://ast-grep.github.io/
- Go patterns catalog: https://ast-grep.github.io/catalog/go/
- Pattern syntax guide: https://ast-grep.github.io/guide/pattern-syntax.html
