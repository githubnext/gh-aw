# Understanding Firewall Logs: Why 0.0.0.0 Appears

## Summary

**TL;DR**: Seeing `0.0.0.0:0` in firewall logs is **normal and expected** for blocked requests. It indicates that the firewall successfully prevented the connection before it could reach an actual destination IP address.

## What is 0.0.0.0 in Firewall Logs?

When you review firewall logs from GitHub Agentic Workflows, you may see entries like this:

```
1761332533.789 172.30.0.20:35291 blocked-domain.example.com:443 0.0.0.0:0 1.1 CONNECT 403 NONE_NONE:HIER_NONE blocked-domain.example.com:443 "-"
```

The `0.0.0.0:0` appears in the **destination IP:port** field (4th field).

## Why Does This Happen?

When the firewall **blocks** a request:

1. The agent attempts to connect to a domain (e.g., `blocked-domain.example.com`)
2. The firewall checks if the domain is in the allowed list
3. **The domain is NOT allowed** → firewall blocks the request immediately
4. Since the connection was denied **before DNS resolution** and routing, there is no real destination IP
5. The firewall records `0.0.0.0:0` as a placeholder for "no destination"

## Contrast with Allowed Requests

Compare a **blocked** request with an **allowed** request:

**Blocked Request** (firewall denies immediately):
```
1761332533.789 172.30.0.20:35291 blocked.example.com:443 0.0.0.0:0 1.1 CONNECT 403 NONE_NONE:HIER_NONE blocked.example.com:443 "-"
```
- Destination IP: `0.0.0.0:0` ❌ (no real destination)
- Status: `403` (Forbidden)
- Decision: `NONE_NONE:HIER_NONE` (connection denied)

**Allowed Request** (firewall permits, connection established):
```
1761332530.474 172.30.0.20:35288 api.github.com:443 140.82.112.22:443 1.1 CONNECT 200 TCP_TUNNEL:HIER_DIRECT api.github.com:443 "-"
```
- Destination IP: `140.82.112.22:443` ✅ (real GitHub IP)
- Status: `200` (OK)
- Decision: `TCP_TUNNEL:HIER_DIRECT` (connection tunneled)

## Log Field Structure

Firewall logs follow this format:
```
timestamp client_ip:port domain dest_ip:port proto method status decision url user_agent
```

| Field | Example (Allowed) | Example (Blocked) | Notes |
|-------|-------------------|-------------------|-------|
| timestamp | `1761332530.474` | `1761332533.789` | Unix timestamp |
| client_ip:port | `172.30.0.20:35288` | `172.30.0.20:35291` | Agent's IP |
| domain | `api.github.com:443` | `blocked.example.com:443` | Target domain |
| **dest_ip:port** | `140.82.112.22:443` | **`0.0.0.0:0`** | Real IP vs. no destination |
| proto | `1.1` | `1.1` | HTTP version |
| method | `CONNECT` | `CONNECT` | HTTP method |
| status | `200` | `403` | HTTP status |
| decision | `TCP_TUNNEL:HIER_DIRECT` | `NONE_NONE:HIER_NONE` | Proxy decision |
| url | `api.github.com:443` | `blocked.example.com:443` | Request URL |
| user_agent | `"-"` | `"-"` | User agent string |

## This is Security Working Correctly

**`0.0.0.0:0` in firewall logs is a good sign!** It means:

✅ The firewall is **actively blocking** unauthorized domains  
✅ The connection was **prevented early** (before DNS resolution)  
✅ No data leaked to the blocked domain  
✅ The security policy is being enforced  

## How to Identify Blocked vs. Allowed Requests

Look at these indicators:

### Blocked Requests
- Destination IP: `0.0.0.0:0`
- Status: `403` or `407`
- Decision contains: `NONE_NONE`, `TCP_DENIED`

### Allowed Requests
- Destination IP: Real IP address (e.g., `140.82.112.22:443`)
- Status: `200`, `206`, or `304`
- Decision contains: `TCP_TUNNEL`, `TCP_HIT`, `TCP_MISS`

## Firewall Log Analysis

You can analyze firewall logs using:

```bash
# View logs for a specific workflow run
gh aw logs --run-id <run-id>

# Audit a workflow run with firewall analysis
gh aw audit <run-id>
```

The analysis will show:
- Total requests made
- Number of allowed requests
- Number of blocked requests
- Lists of allowed and blocked domains

## Common Questions

### Q: Is the firewall blocking my legitimate traffic?

**A**: Check the domain in the firewall log. If you need to allow it, add it to your workflow's `network.allowed` configuration:

```yaml
network:
  allowed:
    - defaults           # Common package managers
    - github            # GitHub APIs
    - your-domain.com   # Your custom domain
```

### Q: Should I be concerned about 0.0.0.0 entries?

**A**: No! These entries show the firewall is working correctly. Only investigate if:
1. A domain you **expect to be allowed** shows `0.0.0.0` (check your `network.allowed` config)
2. Your workflow fails due to a blocked domain (add it to allowed list)

### Q: How do I allow more domains?

**A**: Update your workflow's `network.allowed` configuration:

```yaml
network:
  allowed:
    - defaults        # Includes npm, PyPI, Docker Hub, etc.
    - github          # GitHub APIs
    - node            # Node.js specific domains
    - python          # Python specific domains
    - example.com     # Custom domain
```

See [Network Configuration Guide](https://githubnext.github.io/gh-aw/guides/network-configuration/) for details.

## Related Documentation

- [Network Permissions Reference](https://githubnext.github.io/gh-aw/reference/network/)
- [Firewall Configuration](https://githubnext.github.io/gh-aw/reference/sandbox/)
- [Troubleshooting Network Issues](https://githubnext.github.io/gh-aw/troubleshooting/common-issues/)

## Summary

**Key Takeaway**: `0.0.0.0:0` in firewall logs is **not an error** - it's confirmation that your network security is working as designed. The firewall blocked an unauthorized connection before it could establish a real destination, which is exactly what it should do for security.
