# 🎭 Playwright Browser Automation

Playwright enables browser automation in agentic workflows, providing powerful capabilities for web testing, data extraction, accessibility analysis, and automated user interactions.

## Overview

Playwright runs in a secure containerized environment with configurable network restrictions, making it safe for automated web interactions while maintaining security boundaries.

```yaml
tools:
  playwright:
    docker_image_version: "v1.41.0"
    allowed_domains: ["github.com", "*.github.com"]
```

## Configuration

### Basic Setup

```yaml
engine: claude

tools:
  playwright:
    docker_image_version: "latest"  # Optional: specify version
    allowed_domains: ["example.com"] # Required: specify allowed domains
```

### Docker Image Versions

Specify the Playwright Docker image version from [Microsoft Container Registry](https://mcr.microsoft.com/en-us/product/playwright/about):

```yaml
tools:
  playwright:
    docker_image_version: "v1.41.0"  # Specific version
    # docker_image_version: "latest"  # Latest stable (default)
```

**Supported Versions**: Any valid tag from `mcr.microsoft.com/playwright`

### Domain Restrictions

**Security Requirement**: Always specify `allowed_domains` to control network access:

```yaml
tools:
  playwright:
    allowed_domains: 
      - "github.com"           # Specific domain
      - "*.github.com"         # Wildcard subdomain
      - "localhost"            # Local development
      - "127.0.0.1"           # Local IP
```

**Security Notes**:
- Domain restrictions are enforced at the container level
- Wildcards (`*`) are supported for subdomains
- Always use the minimal set of domains required
- Default behavior blocks all domains if not specified

### Network Integration

Playwright can work with the network permissions system for broader ecosystem access:

```yaml
network:
  allowed:
    - defaults      # GitHub API access
    - playwright    # Playwright ecosystem domains
    
tools:
  playwright:
    docker_image_version: "v1.41.0"
    allowed_domains: ["github.com", "*.example.com"]
```

**Playwright Ecosystem Domains**:
- `playwright.download.prss.microsoft.com` - Download resources
- `cdn.playwright.dev` - CDN assets

## Capabilities

### Browser Engines

Playwright supports multiple browser engines:

- **Chromium**: Modern web standards, good for most automation
- **Firefox**: Gecko engine testing and compatibility
- **Safari/WebKit**: Safari-specific behavior testing

### Core Automation Features

**Page Navigation**:
```yaml
# Example workflow task
1. Navigate to https://example.com
2. Wait for page to load completely
3. Take screenshot of current state
```

**Element Interaction**:
- Click buttons, links, and interactive elements
- Fill forms and input fields
- Keyboard navigation and shortcuts
- Mouse hover and drag operations

**Content Extraction**:
- Extract text content from elements
- Get page titles, URLs, and metadata  
- Access aria labels and accessibility information
- Retrieve network request/response data

**Visual Testing**:
- Full-page screenshots
- Element-specific screenshots
- Compare visual differences
- Responsive design testing

### Accessibility Analysis

Playwright excels at accessibility testing and analysis:

**Accessibility Tree Scanning**:
```yaml
# Analyze page accessibility structure
1. Navigate to target page
2. Extract accessibility tree information
3. Check for missing alt text, aria labels
4. Validate heading hierarchy
5. Test keyboard navigation paths
```

**WCAG Compliance Checks**:
- Color contrast ratio analysis
- Focus indicator visibility
- Screen reader compatibility
- Keyboard accessibility validation

### Mobile and Responsive Testing

```yaml
# Mobile emulation capabilities
1. Emulate iPhone, Android, or custom viewport
2. Test touch interactions and gestures
3. Validate responsive design breakpoints
4. Check mobile-specific accessibility features
```

## Container Environment

### Security Features

**Containerized Execution**:
- Runs in isolated Docker container
- Network access restricted by domain allowlist
- No access to host file system
- Automatic cleanup after execution

**Container Specifications**:
- **Base Image**: `mcr.microsoft.com/playwright:{version}`
- **Memory**: `--shm-size=2gb` for browser stability
- **Capabilities**: `--cap-add=SYS_ADMIN` for Chrome sandbox
- **Network**: Isolated with domain-specific access

### Runtime Environment

**Environment Variables**:
- `PLAYWRIGHT_ALLOWED_DOMAINS`: Comma-separated domain list
- `PLAYWRIGHT_BLOCK_ALL_DOMAINS`: Block all when no domains specified
- Standard Playwright environment variables supported

## Example Workflows

### Basic Web Screenshot

```yaml
---
engine: claude

tools:
  playwright:
    docker_image_version: "v1.41.0"
    allowed_domains: ["github.com"]

safe-outputs:
  create-issue:
    title-prefix: "[Screenshot] "
    max: 1
---

# Website Screenshot Capture

Navigate to GitHub repository and capture a screenshot for documentation.

1. Navigate to https://github.com/githubnext/gh-aw
2. Wait for page to fully load
3. Take a full-page screenshot
4. Save screenshot with timestamp
```

### Accessibility Audit

```yaml
---
engine: claude

tools:
  playwright:
    docker_image_version: "latest"
    allowed_domains: ["*.github.com"]

safe-outputs:
  create-issue:
    title-prefix: "[Accessibility] "
    labels: [accessibility, wcag, automation]
    max: 3
---

# Accessibility Tree Analysis

Scan website accessibility tree and report issues.

1. Navigate to target website
2. Extract complete accessibility tree
3. Analyze for WCAG 2.1 compliance issues:
   - Missing alt text on images
   - Improper heading hierarchy  
   - Missing form labels
   - Insufficient color contrast
   - Missing ARIA landmarks
4. Create detailed issue for each problem found
```

### Form Testing

```yaml
---
engine: claude

tools:
  playwright:
    docker_image_version: "v1.41.0"
    allowed_domains: ["localhost", "127.0.0.1"]
---

# Form Interaction Testing

Test form functionality and validation.

1. Navigate to application form page
2. Fill out form fields with test data
3. Test form validation (empty fields, invalid data)
4. Submit form and verify success/error handling
5. Test keyboard navigation through form
6. Verify accessibility of form elements
```

## Troubleshooting

### Common Issues

**Container Startup Failures**:
- Verify Docker image version exists
- Check domain allowlist configuration
- Ensure sufficient memory allocation

**Network Access Denied**:
- Verify domain in `allowed_domains` list
- Check wildcard syntax for subdomains
- Confirm network permissions if using ecosystem domains

**Browser Launch Failures**:
- Container may need additional capabilities
- Memory constraints (`--shm-size=2gb` is default)
- Chrome sandbox issues (addressed by `SYS_ADMIN` capability)

### Debug Mode

Enable verbose logging for troubleshooting:

```yaml
engine:
  id: claude
  env:
    DEBUG: "playwright:*"
    PLAYWRIGHT_DEBUG: "1"
```

### Performance Optimization

**Faster Execution**:
- Use specific Docker image versions (avoid `latest`)
- Minimize domain allowlist to reduce DNS overhead
- Pin browser engine for consistent performance

**Resource Management**:
- Monitor container memory usage
- Use page timeouts to prevent hanging
- Clean up resources after each operation

## Security Considerations

### Domain Allowlisting

**Best Practices**:
- Always specify minimum required domains
- Use specific domains over wildcards when possible
- Regularly audit domain requirements
- Document why each domain is needed

**Risk Mitigation**:
- Container isolation prevents host access
- Network restrictions limit data exfiltration
- Automatic cleanup prevents persistence
- Audit logs track domain access attempts

### Safe Outputs Integration

Combine with safe outputs for automated issue reporting:

```yaml
safe-outputs:
  create-issue:
    title-prefix: "[Playwright] "
    labels: [automation, testing]
    max: 5  # Limit issue creation
```

## Related Documentation

- [Tools Configuration](tools.md) - Complete tools setup guide
- [Safe Outputs](safe-outputs.md) - Automated issue and PR creation
- [Network Permissions](frontmatter.md#network-permissions) - Network access control
- [Workflow Structure](workflow-structure.md) - Organizing workflows
- [Frontmatter Reference](frontmatter.md) - All configuration options

## Examples Repository

See complete examples in the [Agentics Repository](https://github.com/githubnext/agentics):

- **Accessibility audits**: WCAG compliance scanning
- **Visual regression**: Screenshot comparison workflows
- **Form testing**: Automated form interaction testing
- **Performance testing**: Page load and interaction timing
- **Mobile testing**: Responsive design validation