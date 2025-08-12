---
tools:
  claude:
    allowed:
      Bash:
        - "echo:*"
---

### GitHub Actions Workflow Commands for Structured Output

You can use GitHub Actions workflow commands to generate structured error messages and annotations in your workflow output. These commands create proper annotations in the GitHub Actions UI and show up in pull request checks.

**Available GitHub Actions Workflow Commands:**

1. **Debug Messages** - For detailed information useful for troubleshooting:
   ```bash
   echo "::debug::This is a debug message"
   ```

2. **Notice Messages** - For important information that users should be aware of:
   ```bash
   echo "::notice::This is an informational notice"
   ```

3. **Warning Messages** - For non-fatal issues that should be reviewed:
   ```bash
   echo "::warning::This is a warning message"
   ```

4. **Error Messages** - For critical issues that need immediate attention:
   ```bash
   echo "::error::This is an error message"
   ```

**Enhanced Commands with File Annotations:**
You can also specify file locations for more precise error reporting:

```bash
echo "::error file=filename.js,line=10,col=5::Error found in filename.js at line 10, column 5"
echo "::warning file=package.json,line=15::Deprecated dependency found in package.json"
echo "::notice file=README.md::Documentation updated"
```

**Best Practices for Workflow Error Reporting:**

- Use `::error::` for critical issues that prevent workflow completion
- Use `::warning::` for potential problems that don't break functionality  
- Use `::notice::` for important status updates and successful operations
- Use `::debug::` for detailed diagnostic information
- Include file, line, and column annotations when possible to help developers locate issues quickly
- Keep messages concise but descriptive
- Use these commands at key points in your workflow to provide clear feedback

**Example Usage in Context:**
```bash
# Before a critical operation
echo "::notice::Starting dependency analysis for ${{ github.repository }}"

# After finding an issue
echo "::warning file=go.mod,line=5::Outdated dependency detected: golang.org/x/text"

# On successful completion
echo "::notice::Analysis completed successfully - found 3 issues to review"

# On error
echo "::error::Failed to compile workflow: syntax error in frontmatter"
```

These workflow commands will appear as annotations in the GitHub Actions UI and can be seen in pull request checks, making it easier for developers to understand and act on issues found by your agentic workflow.