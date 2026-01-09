# Gateway Logs Audit - Quick Reference

## Purpose

This PR implements comprehensive infrastructure for auditing and validating MCP gateway logs in smoke-copilot agentic workflow runs.

## What Was Done

### 1. Documentation
- **`docs/AUDIT_GATEWAY_LOGS.md`**: Complete guide for auditing gateway logs
  - Step-by-step audit process
  - Expected artifact structure
  - Troubleshooting common issues
  - Validation checklist

### 2. Automation
- **`scripts/audit-smoke-copilot.sh`**: Automated audit script
  - Finds recent smoke-copilot runs
  - Downloads and analyzes artifacts
  - Validates gateway log presence
  - Reports errors and issues

### 3. Test Coverage
- **`pkg/cli/audit_gateway_logs_test.go`**: Comprehensive test suite
  - `TestAuditWithGatewayLogs`: Normal operation validation
  - `TestAuditWithGatewayErrors`: Error detection and reporting
  - `TestAuditWithoutGatewayLogs`: Direct mode handling

### 4. Summary Report
- **`GATEWAY_LOGS_AUDIT_SUMMARY.md`**: Executive summary
  - Infrastructure validation findings
  - Test results
  - Recommendations
  - Next steps

## Key Findings

✅ **Infrastructure**: Gateway logs are properly captured in `/tmp/gh-aw/mcp-logs/` and uploaded as artifacts

✅ **Audit Command**: The `gh-aw audit` command correctly downloads and preserves gateway logs

✅ **Test Coverage**: All scenarios validated (normal operation, errors, direct mode)

✅ **Documentation**: Complete guides available for auditing and troubleshooting

## Quick Start

### Run Automated Audit
```bash
./scripts/audit-smoke-copilot.sh
```

### Manual Audit
```bash
# List recent smoke-copilot runs
gh run list --repo githubnext/gh-aw --workflow=smoke-copilot.md --limit=10

# Audit a specific run
./gh-aw audit <run-id> --output .github/aw/logs --parse -v

# Check for gateway logs
ls -la .github/aw/logs/run-<run-id>/mcp-logs/gateway-*.log
```

### Run Tests
```bash
# Run gateway log tests
go test -v -run "TestAudit.*Gateway" ./pkg/cli/

# Expected output:
# === RUN   TestAuditWithGatewayLogs
# --- PASS: TestAuditWithGatewayLogs (0.00s)
# === RUN   TestAuditWithGatewayErrors
# --- PASS: TestAuditWithGatewayErrors (0.00s)
# === RUN   TestAuditWithoutGatewayLogs
# --- PASS: TestAuditWithoutGatewayLogs (0.00s)
# PASS
```

## Expected Gateway Log Format

Gateway logs use structured JSON:
```json
{
  "timestamp": "2026-01-09T04:00:00Z",
  "level": "info",
  "msg": "Request completed",
  "method": "tools/call",
  "server": "github",
  "request_id": "abc-123",
  "duration_ms": 42,
  "status": "success"
}
```

## Common Issues

### Gateway Logs Missing
- Check if gateway is configured in workflow
- Verify `mcp-gateway` feature flag is enabled
- Ensure workflow completed execution phase

### Gateway Errors Found
- **Connection failures**: Check MCP server container status
- **Auth failures**: Verify secrets and API keys
- **Timeouts**: Increase `toolTimeout` or investigate slow responses

## Related Documentation

- [Audit Guide](docs/AUDIT_GATEWAY_LOGS.md) - Complete audit procedures
- [Audit Summary](GATEWAY_LOGS_AUDIT_SUMMARY.md) - Findings and recommendations
- [Artifacts Spec](specs/artifacts.md) - Artifact structure reference
- [MCP Gateway Spec](docs/src/content/docs/reference/mcp-gateway.md) - Gateway configuration

## Testing Status

✅ All 3 gateway log tests passing
✅ No breaking changes to existing functionality
✅ Ready for production use

---

**Date**: 2026-01-09  
**PR**: #<number>  
**Author**: GitHub Copilot Agent
