# 🔒 MCP Network Security and Permissions Testing

This document provides comprehensive guidance on MCP (Model Context Protocol) network security, including proxy configuration, domain whitelisting, and network isolation testing.

## Overview

MCP network permissions ensure that agentic workflows can only access explicitly allowed external services, providing strong network isolation and security controls. This is implemented through proxy-based filtering with domain whitelisting.

## Network Security Architecture

### Core Components

1. **Squid Proxy**: Provides network filtering and access control
2. **Domain Whitelist**: Configurable list of allowed domains
3. **Network Isolation**: Container-based network segmentation
4. **Security Monitoring**: Comprehensive access logging and auditing

### Security Controls

- **Default Deny Policy**: All network access blocked by default
- **Domain Whitelisting**: Only explicitly allowed domains are accessible
- **Port Restrictions**: Limited to standard HTTP (80) and HTTPS (443) ports
- **Protocol Filtering**: Proper CONNECT method restrictions for HTTPS
- **Privacy Protection**: User-Agent and referrer headers stripped
- **No Caching**: Prevents data leakage through cached content

## Quick Start

### Setting Up Network Testing

1. **Start the proxy infrastructure**:
   ```bash
   cd docker
   docker-compose up -d
   ```

2. **Test network permissions**:
   ```bash
   # Test with default configuration
   gh aw network-test --urls https://example.com,https://api.github.com
   
   # Test with custom proxy
   gh aw network-test --proxy-host localhost --proxy-port 3128 \
     --domains-file ./docker/squid/allowed_domains.txt \
     --urls https://example.com,https://httpbin.org,https://malicious-example.com
   ```

3. **Validate configuration**:
   ```bash
   gh aw network-validate --domains-file ./docker/squid/allowed_domains.txt
   ```

## Proxy Configuration

### Squid Configuration (`docker/squid/squid.conf`)

The Squid proxy is configured with security-first principles:

```squid
# Domain whitelist - critical security control
acl allowed_domains dstdomain "/etc/squid/allowed_domains.txt"

# CRITICAL: Block all non-whitelisted domains
http_access deny !allowed_domains

# Deny requests to unsafe ports
http_access deny !Safe_ports

# Deny CONNECT to non-SSL ports
http_access deny CONNECT !SSL_ports
```

### Domain Whitelist (`docker/squid/allowed_domains.txt`)

Configure allowed domains in the whitelist file:

```
# Add domains that workflows are allowed to access
example.com
api.trusted-service.com
github.com
```

### Key Security Features

- **Whitelist-based access control**: Only explicitly allowed domains are accessible
- **Port restrictions**: Limited to HTTP (80) and HTTPS (443)
- **Protocol filtering**: Proper CONNECT method restrictions
- **Privacy protection**: Headers stripped to prevent information leakage
- **No caching**: Prevents data persistence and leakage
- **Comprehensive logging**: All access attempts logged for monitoring

## Network Testing Commands

### Basic Network Testing

Test connectivity to specific URLs:

```bash
gh aw network-test --urls https://example.com,https://blocked-site.com
```

### Advanced Testing with Proxy

Test through configured proxy with domain restrictions:

```bash
gh aw network-test \
  --proxy-host localhost \
  --proxy-port 3128 \
  --domains-file ./docker/squid/allowed_domains.txt \
  --urls https://example.com,https://httpbin.org,https://api.github.com,https://malicious-example.com \
  --timeout 30s \
  --verbose
```

### Configuration Validation

Validate proxy and domain configuration files:

```bash
# Validate domains file
gh aw network-validate --domains-file ./docker/squid/allowed_domains.txt

# Validate proxy configuration
gh aw network-validate --config-file ./docker/squid/squid.conf --verbose
```

## Test Result Analysis

### Expected Results

For properly configured network isolation:

- ✅ **Allowed domains**: Should be accessible (HTTP 200/success)
- ❌ **Blocked domains**: Should be inaccessible (network error/timeout)
- 📊 **Consistent behavior**: Results should match domain whitelist

### Sample Test Output

```
=== Network Permission Test Analysis ===
Total Tests: 4
Allowed Domains: 2
Blocked Domains: 2
Successful Connections: 2
Failed Connections: 2

=== Detailed Results ===
✅ ALLOWED & CONNECTED - https://example.com (HTTP 200) [245ms]
✅ ALLOWED & CONNECTED - https://httpbin.org/json (HTTP 200) [312ms]
❌ BLOCKED - https://api.github.com - Error: network timeout [30s]
❌ BLOCKED - https://malicious-example.com - Error: connection refused [156ms]
```

### Interpreting Results

- **✅ ALLOWED & CONNECTED**: Domain is whitelisted and accessible (expected)
- **❌ BLOCKED**: Domain is not whitelisted and blocked (expected)
- **⚠️ UNEXPECTED SUCCESS**: Blocked domain was accessible (security issue)
- **⚠️ ALLOWED BUT FAILED**: Whitelisted domain failed to connect (configuration issue)

## Security Best Practices

### 1. Principle of Least Privilege

Only whitelist domains that are absolutely necessary for workflow functionality:

```
# Good: Specific, necessary domains
api.github.com
api.slack.com

# Avoid: Overly broad or unnecessary domains
*.com
social-media-site.com
```

### 2. Regular Security Audits

- Review domain whitelist monthly
- Monitor proxy access logs for suspicious activity
- Validate network isolation testing regularly
- Update security configurations as needed

### 3. Monitoring and Alerting

Set up monitoring for:
- Unusual domain access patterns
- Failed access attempts to blocked domains
- Proxy service health and availability
- Configuration changes to whitelist

### 4. Defense in Depth

Network isolation is one layer of security. Also implement:
- Input validation in workflows
- Secure secret management
- Regular dependency updates
- Code review processes

## Troubleshooting

### Common Issues

#### 1. Proxy Connection Failures

**Symptoms**: Network tests fail to connect to proxy
```
Error: Failed to connect to proxy http://localhost:3128
```

**Solutions**:
- Verify proxy container is running: `docker-compose ps`
- Check proxy health: `docker-compose exec squid-proxy squid -k check`
- Verify port binding: `netstat -ln | grep 3128`
- Review proxy logs: `docker-compose logs squid-proxy`

#### 2. Unexpected Domain Access

**Symptoms**: Blocked domains are accessible
```
⚠️ UNEXPECTED SUCCESS - https://blocked-site.com (HTTP 200)
```

**Solutions**:
- Verify domain whitelist configuration
- Check for DNS resolution bypassing proxy
- Review Squid ACL configuration
- Confirm containers are using proxy settings

#### 3. Allowed Domains Blocked

**Symptoms**: Whitelisted domains fail to connect
```
⚠️ ALLOWED BUT FAILED - https://example.com - Error: connection refused
```

**Solutions**:
- Verify domain spelling in whitelist
- Check DNS resolution: `nslookup example.com`
- Test direct connectivity outside proxy
- Review Squid access logs for denial reasons

### Debugging Commands

```bash
# Check proxy container status
docker-compose ps

# View proxy logs
docker-compose logs squid-proxy

# Test direct proxy connectivity
curl -x http://localhost:3128 https://example.com

# Validate domains file syntax
gh aw network-validate --domains-file ./docker/squid/allowed_domains.txt

# Run verbose network tests
gh aw network-test --verbose --urls https://example.com
```

## Integration with MCP Workflows

### Configuring MCP Servers with Network Restrictions

When configuring MCP servers in workflows, consider network access requirements:

```yaml
---
tools:
  external-api:
    mcp:
      type: http
      url: "https://api.trusted-service.com/mcp"  # Must be in whitelist
      headers:
        Authorization: "${secrets.API_TOKEN}"
    allowed: ["query_data"]
---

# Workflow content that uses external-api
```

### Network Security in Production

For production deployments:

1. **Isolate MCP containers**: Use dedicated networks for MCP servers
2. **Restrict proxy access**: Limit proxy usage to MCP containers only
3. **Monitor all traffic**: Log and analyze all network requests
4. **Regular updates**: Keep proxy and container images updated
5. **Backup configurations**: Maintain versioned configuration backups

## Related Documentation

- [MCP Integration Guide](mcps.md) - Complete MCP configuration reference
- [Security Notes](security-notes.md) - General security guidelines
- [Tools Configuration](tools.md) - Available tools and configurations

## External Resources

- [Squid Proxy Documentation](http://www.squid-cache.org/Doc/)
- [Docker Compose Reference](https://docs.docker.com/compose/)
- [Model Context Protocol Specification](https://github.com/modelcontextprotocol/specification)