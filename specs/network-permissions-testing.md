# Network Permissions Testing

## Overview

This document describes the network permissions testing infrastructure for gh-aw agentic workflows, specifically focusing on how the Agent Workflow Firewall (AWF) enforces domain restrictions for MCP tools.

## Background

The gh-aw system uses a multi-layered security approach to control network access from agentic workflows:

1. **Agent Workflow Firewall (AWF)**: Containerized firewall using Squid proxy for HTTP/HTTPS traffic filtering
2. **Network Permissions Configuration**: YAML-based domain allow/block lists in workflow frontmatter
3. **MCP Tool Integration**: Tools like `web-fetch` route through the firewall proxy

## Test Workflow

### Location

`.github/workflows/test-mcp-network-permissions.md`

### Purpose

Validates that the AWF correctly enforces network permissions by:
- Allowing access only to explicitly permitted domains
- Blocking all other domain access attempts
- Properly integrating with MCP fetch tool

### Configuration

```yaml
network:
  allowed:
    - example.com

sandbox:
  agent: awf  # Enable Agent Workflow Firewall

tools:
  web-fetch:    # MCP fetch tool for network access testing
```

### Test Cases

The workflow includes 5 comprehensive test cases:

| Test | Domain | Expected Result | Purpose |
|------|--------|-----------------|---------|
| 1 | `example.com` | ✅ **ALLOWED** | Validate allowed domain access works |
| 2 | `httpbin.org` | ❌ **BLOCKED** | Validate non-listed domains are blocked |
| 3 | `api.github.com` | ❌ **BLOCKED** | Validate GitHub API blocked despite GitHub MCP access |
| 4 | `www.google.com` | ❌ **BLOCKED** | Validate major public sites are blocked |
| 5 | `malicious-example.com` | ❌ **BLOCKED** | Validate suspicious domains are blocked |

## Security Architecture

### Firewall Components

#### Squid Proxy Configuration
- **Whitelist-based**: Default deny with explicit allow policy
- **Port**: 3128
- **SSL/HTTPS Support**: Ports 80 and 443
- **Caching**: Disabled for security
- **Privacy**: Headers stripped (`forwarded_for delete`, `via off`)

#### Container Isolation
- MCP containers run in isolated Docker environment
- All network traffic forced through Squid proxy
- No direct internet access
- Health checks monitor proxy availability

#### Domain Allow List
The allow list is generated from:
1. Engine-specific default domains (Copilot, Claude, Codex)
2. User-specified domains in `network.allowed`
3. Ecosystem identifiers (e.g., `node`, `python`, `github`)
4. HTTP MCP server domains (auto-detected)

### Security Strengths

✅ **Proper Network Isolation**: MCP containers use proxy-only access  
✅ **Whitelist-based Approach**: Default deny with explicit allow  
✅ **No Direct Internet Access**: All traffic forced through proxy  
✅ **Health Checks**: Proxy container properly monitored  
✅ **No Caching**: Prevents data leakage between requests  
✅ **Header Stripping**: Protects privacy

### Domain Resolution

The system supports several domain configuration methods:

**1. Explicit Domains**
```yaml
network:
  allowed:
    - example.com
    - api.github.com
```

**2. Ecosystem Identifiers**
```yaml
network:
  allowed:
    - defaults  # Basic infrastructure domains
    - node      # NPM registry, Node.js domains
    - python    # PyPI, Python package domains
    - github    # GitHub.com, API, raw.githubusercontent.com
```

**3. Wildcard Patterns**
```yaml
network:
  allowed:
    - "*.example.com"  # Matches api.example.com, test.example.com, etc.
```

**4. Auto-detected HTTP MCP Domains**
When HTTP MCP servers are configured, their domains are automatically added:
```yaml
tools:
  tavily:
    type: http
    url: https://mcp.tavily.com/mcp/
# mcp.tavily.com is automatically allowed
```

## Network Permissions Validation

### Compilation Time

The compiler validates network configurations during workflow compilation:

1. **Schema Validation**: Network configuration matches expected format
2. **Domain Expansion**: Ecosystem identifiers expanded to domain lists
3. **HTTP MCP Detection**: Domains extracted from HTTP MCP tool URLs
4. **Firewall Configuration**: AWF arguments generated with merged domain list

### Runtime Enforcement

During workflow execution:

1. **Proxy Setup**: Squid proxy configured with allow list
2. **Container Networking**: MCP containers connected to proxy network
3. **Traffic Routing**: HTTP/HTTPS requests routed through proxy
4. **Domain Matching**: Proxy checks requests against allow list
5. **Logging**: Access attempts logged for audit trail

## Testing Methodology

### Manual Testing

To test network permissions manually:

```bash
# 1. Compile the test workflow
gh aw compile .github/workflows/test-mcp-network-permissions.md

# 2. Run the workflow
gh workflow run test-mcp-network-permissions.lock.yml

# 3. Monitor execution
gh run list --workflow=test-mcp-network-permissions.lock.yml

# 4. View logs
gh aw logs [run-id]
```

### Automated Testing

The workflow can be triggered automatically:

1. **Workflow Dispatch**: Manual trigger via GitHub UI or CLI
2. **Pull Request Label**: Triggered when PR is labeled with `test-network-permissions`
3. **Scheduled**: Can be configured with `schedule: daily` for continuous validation

### Expected Results

**Success Scenario:**
```
Test 1 (example.com): ✅ PASS - Successfully fetched content
Test 2 (httpbin.org): ✅ PASS - Blocked by firewall (403 Forbidden)
Test 3 (api.github.com): ✅ PASS - Blocked by firewall (403 Forbidden)
Test 4 (google.com): ✅ PASS - Blocked by firewall (403 Forbidden)
Test 5 (malicious-example.com): ✅ PASS - Blocked by firewall (403 Forbidden)

Overall: PASS - Network permissions correctly enforced
```

**Failure Scenarios:**

1. **Firewall Bypass** (Security Vulnerability):
   - Blocked domain returns successful response
   - Indicates firewall not enforcing restrictions
   - **Action**: Immediate security review required

2. **False Positive Blocking**:
   - Allowed domain is blocked
   - Indicates configuration or firewall misconfiguration
   - **Action**: Check domain list, proxy configuration

3. **Tool Failure**:
   - web-fetch tool fails to initialize
   - Indicates MCP server or container issue
   - **Action**: Check MCP logs, container health

## Issue Reporting

When tests fail, the workflow automatically creates a GitHub issue using safe-outputs:

```yaml
safe-outputs:
  create-issue:
    max: 1
    labels: ['security', 'firewall', 'automated-test']
```

The issue includes:
- Workflow run link
- Failed test details
- Security impact assessment
- Full test results table
- Error messages from firewall logs

## Firewall Log Analysis

### Log Location

During workflow execution, Squid access logs are available at:
- Container path: `/var/log/squid/access.log`
- Uploaded as workflow artifact: `squid-logs`

### Log Format

```
timestamp siaddr code/status bytes method URL rfc931 peerstatus/peerhost type
```

### Common Log Entries

**Allowed Access:**
```
1234567890.123 123 TCP_MISS/200 1234 GET https://example.com/ - HIER_DIRECT/93.184.216.34 text/html
```

**Blocked Access:**
```
1234567890.456 1 TCP_DENIED/403 3918 GET https://httpbin.org/ - HIER_NONE/- text/html
```

### Monitoring Recommendations

Monitor for:
- Successful connections to allowed domains (TCP_MISS/200)
- Blocked connection attempts (TCP_DENIED/403)
- Unusual patterns or repeated failures
- Bypass attempts or suspicious behavior

## Configuration Best Practices

### Minimize Attack Surface

Only allow necessary domains:

```yaml
# ❌ BAD - Too permissive
network:
  allowed:
    - "*"  # Allows all domains (disables firewall)

# ✅ GOOD - Specific domains only
network:
  allowed:
    - example.com
    - api.example.com
```

### Use Ecosystem Identifiers

Leverage built-in ecosystem definitions:

```yaml
# ❌ BAD - Manual domain list (incomplete, hard to maintain)
network:
  allowed:
    - registry.npmjs.org
    - npm.pkg.github.com
    - nodejs.org

# ✅ GOOD - Use ecosystem identifier (complete, maintained)
network:
  allowed:
    - node  # Expands to all Node.js/NPM domains
```

### Explicit Domain Patterns

Be specific with wildcards:

```yaml
# ⚠️ CAUTION - Very broad wildcard
network:
  allowed:
    - "*.example.com"  # Matches any.subdomain.example.com

# ✅ BETTER - Specific subdomains
network:
  allowed:
    - "api.example.com"
    - "cdn.example.com"
```

### Document Rationale

Add comments explaining domain requirements:

```yaml
network:
  allowed:
    - defaults       # Required: Base infrastructure
    - node           # Required: NPM package installs
    - example.com    # Required: API access for workflow logic
    - httpbin.org    # Required: HTTP testing endpoints
```

## Related Documentation

- [Network Permissions Reference](https://githubnext.github.io/gh-aw/reference/network/)
- [Agent Workflow Firewall (AWF)](https://github.com/githubnext/gh-aw-firewall)
- [Safe Outputs Guide](https://githubnext.github.io/gh-aw/reference/safe-outputs/)
- [MCP Tool Configuration](https://githubnext.github.io/gh-aw/guides/mcp/)

## Troubleshooting

### Allowed Domain Not Accessible

**Symptoms**: Domain in allow list but requests fail

**Possible Causes:**
1. Domain not properly added to Squid configuration
2. Container networking issue
3. MCP tool not using proxy
4. DNS resolution failure

**Debug Steps:**
```bash
# 1. Check compiled workflow has correct domains
gh aw compile --verbose test-mcp-network-permissions.md

# 2. Check workflow run logs
gh aw logs [run-id]

# 3. Check Squid logs artifact
gh run download [run-id] --name squid-logs

# 4. Verify domain in AWF args
grep "allow-domains" .github/workflows/test-mcp-network-permissions.lock.yml
```

### Blocked Domain Accessible

**Symptoms**: Domain NOT in allow list but request succeeds

**Security Impact**: **CRITICAL** - Firewall bypass vulnerability

**Immediate Actions:**
1. Stop affected workflows
2. Review firewall implementation
3. Check for container escape attempts
4. Audit Squid configuration
5. Review AWF version and known vulnerabilities

**Debug Steps:**
```bash
# Check Squid process and configuration
docker exec [container-id] ps aux | grep squid
docker exec [container-id] cat /etc/squid/squid.conf

# Check iptables rules
docker exec [container-id] iptables -L -n

# Check actual network routes
docker exec [container-id] ip route show
```

### MCP Tool Connection Failures

**Symptoms**: web-fetch tool fails to initialize or times out

**Possible Causes:**
1. Proxy not running or unhealthy
2. Incorrect proxy configuration
3. MCP server startup failure
4. Docker networking issue

**Debug Steps:**
```bash
# Check Docker containers
docker ps -a | grep -E "fetch|squid"

# Check container logs
docker logs [fetch-container-id]
docker logs [squid-proxy-container-id]

# Check proxy health
curl -x http://localhost:3128 https://example.com/
```

## Future Enhancements

### Planned Improvements

1. **Real-time Monitoring**: Dashboard for network activity
2. **Anomaly Detection**: ML-based suspicious pattern detection  
3. **Performance Metrics**: Track proxy overhead and latency
4. **Enhanced Logging**: Structured logs with correlation IDs
5. **Policy Templates**: Pre-defined security profiles (strict, standard, permissive)

### Feature Requests

See [GitHub Issues](https://github.com/githubnext/gh-aw/labels/network-permissions) for active feature requests.

## Security Considerations

### Threat Model

**Protected Against:**
- Unauthorized outbound connections
- Data exfiltration to non-approved domains
- Malicious payload downloads from untrusted sources
- Lateral movement to internal networks

**Not Protected Against:**
- Data exfiltration through allowed domains
- DNS tunneling (requires DNS-level monitoring)
- Timing-based side channels
- Container escape vulnerabilities (requires SRT/AWF hardening)

### Security Boundaries

1. **Container Isolation**: First line of defense
2. **Network Proxy**: Second line (application-level filtering)
3. **Host Firewall**: Third line (network-level filtering)
4. **GitHub Actions**: Fourth line (workflow-level isolation)

### Compliance Notes

The network permissions system helps meet compliance requirements for:
- **Data Residency**: Control where data can be sent
- **Audit Trail**: Log all network access attempts
- **Least Privilege**: Minimize network exposure
- **Defense in Depth**: Multiple security layers

## Conclusion

The network permissions testing infrastructure provides comprehensive validation of AWF's domain restriction capabilities. By systematically testing allowed and blocked domains, we ensure that:

1. Firewall correctly enforces network policies
2. Allowed domains remain accessible
3. Blocked domains cannot bypass restrictions
4. Security configurations are properly validated

Regular execution of the test workflow helps maintain security posture and catch configuration drift or firewall regressions.
