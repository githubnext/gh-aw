Thanks for helping make GitHub safe for everyone.

# Security

GitHub takes the security of our software products and services seriously, including all of the open source code repositories managed through our GitHub organizations, such as [GitHub](https://github.com/GitHub).

Even though [open source repositories are outside of the scope of our bug bounty program](https://bounty.github.com/index.html#scope) and therefore not eligible for bounty rewards, we will ensure that your finding gets passed along to the appropriate maintainers for remediation.

## Reporting Security Issues

If you believe you have found a security vulnerability in any GitHub-owned repository, please report it to us through coordinated disclosure.

**Please do not report security vulnerabilities through public GitHub issues, discussions, or pull requests.**

Instead, please send an email to opensource-security[@]github.com.

Please include as much of the information listed below as you can to help us better understand and resolve the issue:

- The type of issue (e.g., buffer overflow, SQL injection, or cross-site scripting)
- Full paths of source file(s) related to the manifestation of the issue
- The location of the affected source code (tag/branch/commit or direct URL)
- Any special configuration required to reproduce the issue
- Step-by-step instructions to reproduce the issue
- Proof-of-concept or exploit code (if possible)
- Impact of the issue, including how an attacker might exploit the issue

This information will help us triage your report more quickly.

## Policy

See [GitHub's Safe Harbor Policy](https://docs.github.com/en/github/site-policy/github-bug-bounty-program-legal-safe-harbor#1-safe-harbor-terms)

## Security Best Practices

### Template Injection Prevention

When creating GitHub Actions workflows with gh-aw, follow these guidelines to prevent template injection vulnerabilities:

#### Key Rules

- **Never use `envsubst` on data from GitHub expressions** - This allows code execution through shell command expansion
- **Use sed-based substitution with proper escaping** - Treats user input as literal strings
- **Always treat user input as untrusted data** - Issue bodies, PR titles, comments, discussion content, etc.
- **Use placeholders like `__VAR__` for clarity** - Makes template substitution explicit and searchable
- **Pass untrusted data through environment variables** - Not directly in template expressions

#### Secure Substitution Pattern

```yaml
env:
  USER_DATA: ${{ github.event.issue.body }}
run: |
  # Write template with placeholder
  cat << 'PROMPT_EOF' > "$OUTPUT_FILE"
  Content with __USER_DATA__ placeholder
  PROMPT_EOF
  
  # Safe substitution with sed (no shell expansion)
  sed -i "s|__USER_DATA__|${USER_DATA//|/\\|}|g" "$OUTPUT_FILE"
```

#### Unsafe Patterns to Avoid

❌ **Never use template expressions directly in shell commands:**
```yaml
run: |
  echo "User input: ${{ github.event.issue.body }}"  # UNSAFE
```

❌ **Never use envsubst on untrusted data:**
```yaml
run: |
  export USER_DATA="${{ github.event.issue.body }}"
  envsubst < template.txt > output.txt  # UNSAFE
```

#### Safe Context Variables

These GitHub context variables are always safe to use directly:
- `${{ github.actor }}`
- `${{ github.repository }}`
- `${{ github.run_id }}`
- `${{ github.run_number }}`
- `${{ github.sha }}`

#### Untrusted Context Variables

These must always be passed through environment variables:
- `${{ github.event.issue.title }}` / `${{ github.event.issue.body }}`
- `${{ github.event.comment.body }}`
- `${{ github.event.pull_request.title }}` / `${{ github.event.pull_request.body }}`
- `${{ github.event.discussion.title }}` / `${{ github.event.discussion.body }}`
- `${{ github.event.head_commit.message }}`
- `${{ github.head_ref }}` (can be controlled by PR authors)
- `${{ github.ref_name }}` (branch/tag names)
- `${{ steps.*.outputs.* }}` (step outputs may contain user data)

#### Validation Tools

Use these tools to detect template injection vulnerabilities:

```bash
# Compile workflow with security scanning
./gh-aw compile workflow-name --zizmor --actionlint --poutine

# Run security scans
make security-scan
```

#### Additional Resources

- See `examples/secure-templating.md` for a complete reference workflow
- See `specs/template-injection-prevention.md` for detailed vulnerability analysis
- See `DEVGUIDE.md` for secure template substitution guidelines
- [GitHub Actions Security Hardening](https://docs.github.com/en/actions/security-guides/security-hardening-for-github-actions)
- [Understanding Script Injection Risk](https://docs.github.com/en/actions/security-guides/security-hardening-for-github-actions#understanding-the-risk-of-script-injections)

## Software Bill of Materials (SBOM)

We generate Software Bill of Materials (SBOM) for this project to provide complete visibility into the dependency tree, enabling compliance reporting, vulnerability tracking, and supply chain risk assessment.

### SBOM Generation

SBOMs are automatically generated on every release and attached to GitHub releases as downloadable assets.

Both SPDX and CycloneDX formats are generated to ensure compatibility with different compliance and security tools.

### Local SBOM Generation

To generate an SBOM locally, first install [syft](https://github.com/anchore/syft):

```bash
# Install syft
curl -sSfL https://raw.githubusercontent.com/anchore/syft/main/install.sh | sh -s -- -b /usr/local/bin

# Generate SBOM
make sbom
```

This produces two files:
- `sbom.spdx.json` - SBOM in SPDX JSON format
- `sbom.cdx.json` - SBOM in CycloneDX JSON format

### SBOM Contents

The generated SBOMs include:
- All direct and transitive Go dependencies
- Package versions and licenses
- Package hashes for integrity verification
- Dependency relationships
