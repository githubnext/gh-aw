# MCP Network Permissions Testing Infrastructure

This directory contains Docker infrastructure for testing MCP (Model Context Protocol) network permissions and proxy configuration.

## Overview

The testing infrastructure provides:
- **Squid Proxy**: Network filtering with domain whitelisting
- **Test Container**: Alpine-based container for network testing
- **Network Isolation**: Dedicated Docker network for testing
- **Health Checks**: Container health monitoring

## Quick Start

### 1. Start the Infrastructure

```bash
# From the repository root
cd docker
docker-compose up -d
```

### 2. Verify Setup

```bash
# Check container status
docker-compose ps

# Check proxy health
docker-compose exec squid-proxy squid -k check

# View proxy logs
docker-compose logs squid-proxy
```

### 3. Test Network Permissions

```bash
# From the repository root
gh aw network-test \
  --proxy-host localhost \
  --proxy-port 3128 \
  --domains-file ./docker/squid/allowed_domains.txt \
  --urls https://example.com,https://httpbin.org,https://github.com
```

## Configuration Files

### `docker-compose.yml`
- Defines the complete testing infrastructure
- Configures proxy and test containers
- Sets up isolated network for testing
- Includes health checks and dependencies

### `squid/squid.conf`
- Squid proxy configuration with security controls
- Implements domain whitelisting via ACLs
- Provides port and protocol restrictions
- Enables comprehensive access logging

### `squid/allowed_domains.txt`
- Domain whitelist configuration
- Only listed domains are accessible through proxy
- Supports comments and empty lines
- Easy to modify for testing scenarios

## Testing Scenarios

### Scenario 1: Basic Network Isolation

Test that only whitelisted domains are accessible:

```bash
# Should succeed (whitelisted)
curl -x http://localhost:3128 https://example.com

# Should fail (not whitelisted)
curl -x http://localhost:3128 https://github.com
```

### Scenario 2: MCP Container Testing

Test network access from within the MCP test container:

```bash
# Access the test container
docker-compose exec mcp-fetch-container sh

# Test allowed domain (should work)
curl https://example.com

# Test blocked domain (should fail)
curl https://github.com
```

### Scenario 3: Configuration Validation

Validate proxy configuration and domain lists:

```bash
# Validate domains file
gh aw network-validate --domains-file ./squid/allowed_domains.txt

# Test proxy connectivity
gh aw network-test --proxy-host localhost --proxy-port 3128 --urls https://example.com
```

## Security Features

### Network Isolation
- Dedicated Docker network (`mcp-test-network`)
- Isolated subnet (172.20.0.0/16)
- Container-to-container communication only through proxy

### Access Control
- **Default Deny**: All domains blocked by default
- **Whitelist Only**: Only explicitly allowed domains accessible
- **Port Restrictions**: Limited to HTTP (80) and HTTPS (443)
- **Protocol Filtering**: Proper CONNECT method restrictions

### Privacy Protection
- **No Caching**: Prevents data persistence
- **Header Stripping**: User-Agent and referrer headers removed
- **DNS Control**: Uses public DNS (8.8.8.8, 8.8.4.4)

### Monitoring
- **Access Logging**: All requests logged to `/var/log/squid/access.log`
- **Cache Logging**: System events logged to `/var/log/squid/cache.log`
- **Health Checks**: Container health monitoring

## Troubleshooting

### Container Issues

```bash
# Check all container status
docker-compose ps

# View container logs
docker-compose logs squid-proxy
docker-compose logs mcp-fetch-container

# Restart containers
docker-compose restart
```

### Proxy Issues

```bash
# Test proxy directly
curl -x http://localhost:3128 https://example.com

# Check proxy configuration
docker-compose exec squid-proxy squid -k parse

# View proxy access logs
docker-compose exec squid-proxy tail -f /var/log/squid/access.log
```

### Network Issues

```bash
# Check network connectivity
docker network ls
docker network inspect docker_mcp-test-network

# Test DNS resolution
docker-compose exec mcp-fetch-container nslookup example.com

# Check port binding
netstat -ln | grep 3128
```

## Customization

### Adding Allowed Domains

Edit `squid/allowed_domains.txt`:
```
# Add new allowed domains
example.com
api.trusted-service.com
github.com
```

Then restart the proxy:
```bash
docker-compose restart squid-proxy
```

### Modifying Proxy Configuration

Edit `squid/squid.conf` and restart:
```bash
docker-compose restart squid-proxy
```

### Testing Different Scenarios

Create different domain files for testing:
```bash
# Create test-specific domains file
echo -e "example.com\ntest-domain.com" > squid/test_domains.txt

# Test with custom domains file
gh aw network-test \
  --domains-file ./docker/squid/test_domains.txt \
  --urls https://example.com,https://test-domain.com,https://blocked.com
```

## Cleanup

```bash
# Stop and remove containers
docker-compose down

# Remove volumes and networks
docker-compose down -v

# Remove images (optional)
docker-compose down --rmi all
```

## Production Considerations

When adapting this infrastructure for production:

1. **Persistent Logging**: Mount log volumes for persistent storage
2. **Configuration Management**: Use configuration management tools
3. **Monitoring**: Integrate with monitoring and alerting systems
4. **Security Updates**: Regularly update container images
5. **Backup**: Backup configuration files and logs
6. **Access Control**: Restrict proxy access to authorized containers only

## Related Documentation

- [Network Security Guide](../docs/network-security.md) - Comprehensive network security documentation
- [MCP Integration Guide](../docs/mcps.md) - MCP configuration and usage
- [CLI Commands](../docs/commands.md) - Available CLI commands for testing