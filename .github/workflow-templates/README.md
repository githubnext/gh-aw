# Secure Workflow Templates

This directory contains secure workflow templates demonstrating best practices for common workflow patterns in GitHub Agentic Workflows.

## Available Templates

### 1. slash-command.md
**Pattern:** Secure slash command handling from issue/PR comments

**Security Features:**
- User input passed via environment variables (prevents template injection)
- Command validation and allowlisting
- Explicit permission scoping (read-only with safe outputs)
- Repository member verification

**Use Case:** Processing slash commands like `/analyze`, `/summarize`, `/help` from comments

**Example Usage:**
```bash
# Copy template to your workflows directory
cp .github/workflow-templates/slash-command.md .github/workflows/my-command-handler.md

# Edit to customize commands and behavior
# Compile the workflow
gh aw compile .github/workflows/my-command-handler.md
```

### 2. safe-pr-handler.md
**Pattern:** Secure pull request processing

**Security Features:**
- User input passed via environment variables
- Read-only permissions by default
- Safe outputs for write operations
- Fork protection enabled
- Input validation before processing

**Use Case:** Analyzing PRs, providing automated feedback, suggesting labels

**Example Usage:**
```bash
# Copy template to your workflows directory
cp .github/workflow-templates/safe-pr-handler.md .github/workflows/pr-analyzer.md

# Customize analysis logic
# Compile the workflow
gh aw compile .github/workflows/pr-analyzer.md
```

### 3. artifact-upload.md
**Pattern:** Secure artifact handling with validation

**Security Features:**
- Pre-upload secret scanning
- File exclusion patterns for sensitive files
- Size limits validation
- Explicit retention policy
- Checksum verification

**Use Case:** Building and uploading artifacts with security validation

**Example Usage:**
```bash
# Copy template to your workflows directory
cp .github/workflow-templates/artifact-upload.md .github/workflows/build-artifacts.md

# Customize build process and exclusions
# Compile the workflow
gh aw compile .github/workflows/build-artifacts.md
```

## Using Templates

1. **Copy the template** to your `.github/workflows/` directory
2. **Customize** the workflow for your specific needs
3. **Compile** the workflow: `gh aw compile <workflow-name>.md`
4. **Test** the compiled workflow locally or in a test branch
5. **Deploy** by pushing the compiled `.lock.yml` file

## Security Best Practices

All templates follow these security principles:

- ✅ **Template injection prevention** - User input never in expressions
- ✅ **Minimal permissions** - Read-only by default
- ✅ **Safe outputs** - Write operations through validated safe-outputs
- ✅ **Input validation** - All user input validated before use
- ✅ **Secret handling** - No hardcoded secrets, proper masking

## Additional Resources

### Documentation
- [Secure Workflow Authoring Guide](https://githubnext.github.io/gh-aw/guides/workflow-security-guide/)
- [Validation Rules Reference](https://githubnext.github.io/gh-aw/reference/validation-rules/)
- [CI Validation Setup](https://githubnext.github.io/gh-aw/guides/ci-validation-setup/)
- [Security Best Practices](https://githubnext.github.io/gh-aw/guides/security/)

### Internal Specs
- `specs/template-injection-prevention.md` - Template injection patterns
- `specs/github-actions-security-best-practices.md` - Security best practices

### Validation Tools
- `scripts/validate-workflow.sh` - Validate workflow before commit
- `.githooks/pre-commit` - Automatic validation on commit

## Contributing Templates

Have a secure workflow pattern to share? Consider contributing a template:

1. Create a new template following the established patterns
2. Include comprehensive security documentation in comments
3. Add validation checks and error handling
4. Update this README with the new template
5. Submit a pull request

Templates should be:
- **Secure by default** - Follow all security best practices
- **Well-documented** - Clear comments explaining security features
- **Reusable** - Easy to customize for different use cases
- **Validated** - Pass all linting and security scans

## Getting Help

- **Documentation**: https://githubnext.github.io/gh-aw/
- **Issues**: https://github.com/githubnext/gh-aw/issues
- **Discord**: #continuous-ai channel in [GitHub Next Discord](https://gh.io/next-discord)
