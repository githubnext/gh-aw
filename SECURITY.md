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

## Dependency Review

We use automated dependency review to prevent vulnerable or non-compliant dependencies from being introduced through pull requests.

### Pull Request Dependency Checks

The [Dependency Review Action](https://github.com/actions/dependency-review-action) automatically runs on all pull requests to:

- **Block HIGH and CRITICAL vulnerabilities** - PRs that introduce dependencies with known high or critical severity vulnerabilities will fail checks
- **Enforce license compliance** - PRs that introduce dependencies with incompatible licenses (GPL-3.0, AGPL-3.0) will fail checks
- **Provide inline feedback** - Vulnerability and license findings are posted as PR comments for easy review

### Blocked Dependency Changes

PRs will be blocked if they introduce:

1. **High or Critical Vulnerabilities** - Any dependency with a HIGH or CRITICAL severity CVE
2. **Incompatible Licenses** - Dependencies with copyleft licenses incompatible with Apache-2.0:
   - GPL-3.0 (GNU General Public License v3.0)
   - AGPL-3.0 (GNU Affero General Public License v3.0)

### Allowed Licenses

The following permissive licenses are compatible with Apache-2.0 and allowed:
- Apache-2.0, MIT, BSD-2-Clause, BSD-3-Clause
- ISC, 0BSD, CC0-1.0, Unlicense, MPL-2.0

### Handling Blocked Dependencies

If your PR is blocked by dependency review:

1. **For vulnerabilities**: Update to a patched version or find an alternative dependency
2. **For license issues**: Choose a dependency with a compatible license or seek legal review
3. **For false positives**: Document the reasoning and request maintainer override

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
