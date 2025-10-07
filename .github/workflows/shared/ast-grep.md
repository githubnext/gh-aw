---
tools:
  bash: ["ast-grep:*", "sg:*"]
steps:
  - name: Install ast-grep
    run: |
      curl -L https://github.com/ast-grep/ast-grep/releases/latest/download/ast-grep-x86_64-unknown-linux-gnu.zip -o /tmp/ast-grep.zip
      unzip -q /tmp/ast-grep.zip -d /tmp/ast-grep
      sudo mv /tmp/ast-grep/ast-grep /usr/local/bin/
      chmod +x /usr/local/bin/ast-grep
      ast-grep --version
---

## ast-grep Tool Setup

### Using ast-grep

ast-grep (also available as `sg` command) is a powerful structural search and replace tool for code. It uses tree-sitter grammars to parse and search code based on its structure rather than just text patterns.

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
