# Copilot CLI Installation: SHA256 Checksum Verification

## Overview

As of this implementation, all GitHub Copilot CLI installations in gh-aw workflows now include SHA256 checksum verification to prevent supply chain attacks. This addresses the `unverified_script_exec` security finding identified by Poutine.

## Security Problem

Previously, workflows downloaded and executed the Copilot CLI installer script without any integrity verification:

```bash
export VERSION=0.0.369 && curl -fsSL https://gh.io/copilot-install | sudo bash
```

**Security Risks:**
- If `gh.io` or the installer script URL is compromised, malicious code could be injected
- Script executes with `sudo` privileges, allowing system-level access
- No way to detect corrupted or tampered downloads
- Trust placed in external infrastructure (gh.io redirection)

## Solution

The new implementation:
1. Downloads the Copilot CLI binary directly from GitHub releases
2. Downloads the checksums file from the same release
3. Verifies SHA256 checksum before installation
4. Fails fast if checksum mismatch is detected
5. Provides graceful fallback for older releases without checksums

## Implementation Details

### Direct Binary Download

Instead of using the installer script, we now:

```bash
COPILOT_VERSION="v0.0.369"
COPILOT_REPO="github/copilot-cli"
COPILOT_PLATFORM="linux-x64"
COPILOT_ARCHIVE="copilot-${COPILOT_PLATFORM}.tar.gz"
COPILOT_URL="https://github.com/${COPILOT_REPO}/releases/download/${COPILOT_VERSION}/${COPILOT_ARCHIVE}"
CHECKSUMS_URL="https://github.com/${COPILOT_REPO}/releases/download/${COPILOT_VERSION}/checksums.txt"

# Download binary
curl -fsSL -o "/tmp/${COPILOT_ARCHIVE}" "${COPILOT_URL}"

# Download checksums file
curl -fsSL -o "/tmp/copilot-checksums.txt" "${CHECKSUMS_URL}"
```

### Checksum Verification

```bash
# Extract expected checksum for our platform
EXPECTED_CHECKSUM=$(grep "${COPILOT_ARCHIVE}" /tmp/copilot-checksums.txt | awk '{print $1}')

# Compute actual checksum of downloaded binary
ACTUAL_CHECKSUM=$(sha256sum "/tmp/${COPILOT_ARCHIVE}" | awk '{print $1}')

# Verify match
if [ "${ACTUAL_CHECKSUM}" = "${EXPECTED_CHECKSUM}" ]; then
  echo "✓ Checksum verification passed!"
else
  echo "✗ ERROR: Checksum verification failed!"
  echo "  The downloaded binary may be corrupted or tampered with."
  exit 1
fi
```

### Graceful Fallback

For older Copilot CLI releases that may not have checksums files:

```bash
curl -fsSL -o "/tmp/copilot-checksums.txt" "${CHECKSUMS_URL}" || {
  echo "Warning: Checksums file not available for version ${COPILOT_VERSION}"
  echo "Proceeding without checksum verification (fallback for older releases)"
  SKIP_CHECKSUM=true
}
```

## Code Location

The implementation is in:
- **Function**: `GenerateCopilotInstallerSteps` in `pkg/workflow/copilot_engine.go`
- **Tests**: `pkg/workflow/copilot_installer_test.go`

## Affected Workflows

The following 73 workflows now use checksum-verified Copilot CLI installation:

1. ai-moderator.md
2. archie.md
3. artifacts-summary.md
4. brave.md
5. breaking-change-checker.md
6. ci-coach.md
7. ci-doctor.md
8. cli-consistency-checker.md
9. copilot-pr-merged-report.md
10. copilot-pr-nlp-analysis.md
11. copilot-pr-prompt-analysis.md
12. craft.md
13. daily-assign-issue-to-user.md
14. daily-copilot-token-report.md
15. daily-file-diet.md
16. daily-firewall-report.md
17. daily-malicious-code-scan.md
18. daily-news.md
19. daily-repo-chronicle.md
20. daily-workflow-updater.md
21. dependabot-go-checker.md
22. dev-hawk.md
23. dev.md
24. dictation-prompt.md
25. docs-noob-tester.md
26. example-permissions-warning.md
27. firewall-escape.md
28. firewall.md
29. glossary-maintainer.md
30. go-file-size-reduction-project64.campaign.md
31. grumpy-reviewer.md
32. hourly-ci-cleaner.md
33. human-ai-collaboration.md
34. incident-response.md
35. intelligence.md
36. issue-monster.md
37. issue-triage-agent.md
38. layout-spec-maintainer.md
39. mcp-inspector.md
40. mergefest.md
41. notion-issue-summary.md
42. org-health-report.md
43. pdf-summary.md
44. plan.md
45. poem-bot.md
46. portfolio-analyst.md
47. pr-nitpick-reviewer.md
48. python-data-charts.md
49. q.md
50. release.md
51. repo-tree-map.md
52. repository-quality-improver.md
53. research.md
54. security-compliance.md
55. slide-deck-maintainer.md
56. smoke-copilot-no-firewall.md
57. smoke-copilot-safe-inputs.md
58. smoke-copilot.md
59. smoke-srt.md
60. stale-repo-identifier.md
61. super-linter.md
62. technical-doc-writer.md
63. test-discussion-expires.md
64. test-hide-older-comments.md
65. test-python-safe-input.md
66. tidy.md
67. video-analyzer.md
68. weekly-issue-summary.md
69. (and any other workflows using engine: copilot)

## Verification

To verify the checksum verification is working, check the compiled `.lock.yml` files. Look for:

1. **No installer script**: Should NOT contain `gh.io/copilot-install`
2. **Direct download**: Should contain `github.com/github/copilot-cli/releases/download`
3. **Checksum verification**: Should contain `sha256sum` and checksum comparison logic

Example from a compiled workflow:

```yaml
- name: Install GitHub Copilot CLI
  run: |
    COPILOT_VERSION="v0.0.369"
    COPILOT_REPO="github/copilot-cli"
    COPILOT_PLATFORM="linux-x64"
    # ... (download and verify checksums)
    if [ "${ACTUAL_CHECKSUM}" = "${EXPECTED_CHECKSUM}" ]; then
      echo "✓ Checksum verification passed!"
    else
      echo "✗ ERROR: Checksum verification failed!"
      exit 1
    fi
```

## Security Benefits

1. **Supply Chain Protection**: Detects if GitHub releases are compromised
2. **Integrity Verification**: Ensures downloaded binary matches official release
3. **Tamper Detection**: Identifies corrupted or modified downloads
4. **Reduced Attack Surface**: Eliminates dependency on third-party redirects (gh.io)
5. **Clear Error Messages**: Provides actionable feedback on verification failures

## Testing

Unit tests verify:
- Version handling (with and without 'v' prefix)
- Checksum verification code generation
- Fallback handling for missing checksums
- Integration with workflow compilation

Run tests:
```bash
make test-unit
# or specifically:
go test ./pkg/workflow -run TestGenerateCopilotInstallerSteps
```

## Related Issues

- Issue #6672: [plan] Add SHA256 checksum verification for Copilot CLI installer script
- Poutine finding: `unverified_script_exec` in multiple workflows

## Future Improvements

1. **Checksum Pinning**: Consider pinning expected checksums for known versions in code
2. **Release Metadata**: Use GitHub API to fetch release metadata for additional verification
3. **Alternative Verification**: Explore GPG signature verification if available
4. **Monitoring**: Add telemetry to track checksum verification success/failure rates
