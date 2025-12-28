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

## Workflow Security Policy

This project maintains strict security standards for GitHub Actions workflows to protect against supply chain attacks and maintain secure CI/CD pipelines.

### Action Pinning Requirement

**All GitHub Actions must be pinned to full commit SHAs, not tags or branches.**

```yaml
# ✅ REQUIRED: Pin to immutable SHA with version comment
- uses: actions/checkout@b4ffde65f46336ab88eb53be808477a3936bae11 # v4.1.1

# ❌ NOT ALLOWED: Mutable tag or branch references
- uses: actions/checkout@v4
- uses: actions/checkout@main
```

**Rationale**: Tags and branches can be deleted and recreated with malicious code. SHA commits are immutable and provide protection against supply chain attacks.

### Security Update Process

1. **Monitoring**: We use Dependabot to monitor action versions and security advisories
2. **Review**: All action updates are reviewed for security implications before merging
3. **Testing**: Updated workflows are tested in CI before deployment
4. **Documentation**: Security-relevant changes are documented in release notes

### Supported Versions

We provide security updates for:
- The latest major version
- The previous major version for 90 days after a new major release

| Version | Supported          | Security Updates Until |
| ------- | ------------------ | --------------------- |
| 0.x.x   | :white_check_mark: | Current development   |

### Security Standards

Our workflows follow these security standards:

- **Minimal Permissions**: Workflows use the principle of least privilege
- **Template Injection Prevention**: Untrusted input is never used directly in expressions
- **Input Validation**: All external inputs are validated before use
- **Network Isolation**: Workflows restrict network access where applicable
- **Third-Party Verification**: Third-party actions are reviewed before use

For detailed security practices, see:
- [Workflow Security Guidelines](CONTRIBUTING.md#workflow-security-guidelines) - Contributor guide
- [GitHub Actions Security Best Practices](specs/github-actions-security-best-practices.md) - Comprehensive technical guide
- [Template Injection Prevention](specs/template-injection-prevention.md) - Injection attack prevention

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
