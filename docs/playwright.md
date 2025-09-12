# 🎭 Playwright Browser Automation

Playwright enables powerful browser automation in agentic workflows, providing capabilities for web testing, data extraction, accessibility analysis, and automated user interactions in a secure containerized environment.

## Overview

Playwright runs inside a containerized Docker environment with configurable network domain restrictions, making it safe for automated web interactions while maintaining security boundaries. All browser automation happens in an isolated container that cannot access local resources.

## Configuration

### Basic Setup

Add Playwright to your workflow tools configuration:

```yaml
tools:
  playwright:
    docker_image_version: "latest"
    allowed_domains: ["localhost", "127.0.0.1"]
```

### Domain Configuration

Playwright supports fine-grained domain access control through the `allowed_domains` configuration. You can specify individual domains or use ecosystem bundle identifiers:

```yaml
tools:
  playwright:
    allowed_domains: ["defaults", "github", "example.com"]
```

#### Available Domain Bundles

| Bundle | Description | Example Domains |
|--------|-------------|----------------|
| `defaults` | Certificate authorities, package repos | `crl3.digicert.com`, `archive.ubuntu.com` |
| `github` | GitHub services and APIs | `github.com`, `api.github.com`, `objects.githubusercontent.com` |
| `node` | Node.js ecosystem | `registry.npmjs.org`, `nodejs.org` |
| `python` | Python package index | `pypi.org`, `files.pythonhosted.org` |
| `containers` | Container registries | `ghcr.io`, `registry.hub.docker.com` |
| `ai` | AI service providers | `api.openai.com`, `api.anthropic.com` |

#### Domain Syntax

- **Wildcard domains**: `"*.example.com"` matches all subdomains
- **Exact domains**: `"example.com"` matches only the exact domain
- **IP addresses**: `"127.0.0.1"`, `"localhost"`
- **Mixed configuration**: Combine bundles and specific domains

```yaml
tools:
  playwright:
    allowed_domains: ["github", "*.playwright.dev", "example.com"]
```

### Docker Image Versions

Specify the Playwright Docker image version:

```yaml
tools:
  playwright:
    docker_image_version: "v1.41.0"  # or "latest"
```

## Security Model

### Default Restrictions

- **Localhost only**: By default, only `localhost` and `127.0.0.1` are accessible
- **Containerized execution**: All browser automation runs in isolated Docker containers
- **Network isolation**: No access to internal networks or local file system
- **Domain allowlisting**: Explicit domain approval required for external access

### Safe Practices

1. **Minimal domain access**: Only allow domains necessary for your workflow
2. **Use specific domains**: Prefer exact domains over wildcards when possible
3. **Bundle identification**: Use ecosystem bundles for trusted domain sets
4. **Regular review**: Periodically review and update domain allowlists

## Capabilities

### Browser Automation

- **Multi-browser support**: Chromium, Firefox, Safari engines
- **Page navigation**: Navigate to URLs, handle redirects and authentication
- **Element interaction**: Click, type, scroll, drag-and-drop operations
- **Form automation**: Fill forms, select options, upload files

### Data Extraction

- **Content extraction**: Extract text, attributes, and structured data
- **Screenshot capture**: Full page or element-specific screenshots
- **PDF generation**: Convert pages to PDF documents
- **Network monitoring**: Capture and analyze network requests

### Accessibility Analysis

- **Accessibility tree**: Analyze page structure for accessibility compliance
- **WCAG validation**: Check for WCAG 2.1 AA compliance violations
- **Color contrast**: Analyze color contrast ratios for readability
- **Screen reader compatibility**: Test screen reader navigation paths

### Testing Capabilities

- **Visual regression**: Compare screenshots for visual changes
- **Performance monitoring**: Measure page load times and metrics
- **Mobile emulation**: Test responsive designs and mobile layouts
- **Cross-browser testing**: Validate functionality across different browsers

## Example Workflows

### Screenshot Analysis

```yaml
---
on: workflow_dispatch
permissions: read-all
engine:
  id: claude

tools:
  playwright:
    docker_image_version: "latest"
    allowed_domains: ["github.com", "*.github.com"]

safe-outputs:
  create-issue:
    title-prefix: "[Screenshot Analysis] "
    labels: [automation, visual-testing]
    max: 1
---

# Screenshot and Visual Analysis

Take a screenshot of the repository page and analyze it for visual issues or improvements.

## Steps

1. Navigate to the repository page
2. Take a full-page screenshot
3. Analyze the visual layout and design
4. Create an issue if any problems are found
```

### Accessibility Audit

```yaml
---
on: workflow_dispatch
permissions: read-all
engine:
  id: claude

tools:
  playwright:
    docker_image_version: "latest"
    allowed_domains: ["defaults", "github"]

safe-outputs:
  create-issue:
    title-prefix: "[Accessibility] "
    labels: [accessibility, a11y, wcag]
    max: 3
---

# WCAG Accessibility Compliance Audit

Analyze the accessibility tree and check for WCAG 2.1 AA compliance issues.

## Analysis Areas

1. **Semantic Structure**: Check heading hierarchy and landmark usage
2. **Form Accessibility**: Verify proper labeling and error handling
3. **Keyboard Navigation**: Test tab order and keyboard shortcuts
4. **Color Contrast**: Analyze contrast ratios for text readability
5. **Alternative Text**: Check image alt text and descriptions
```

### Performance Monitoring

```yaml
---
on:
  schedule:
    - cron: "0 */6 * * *"  # Every 6 hours
permissions: read-all
engine:
  id: claude

tools:
  playwright:
    docker_image_version: "latest"
    allowed_domains: ["github.com"]
---

# Performance Monitoring

Monitor page load performance and create alerts for performance degradation.

## Metrics to Collect

1. **Load Times**: First contentful paint, largest contentful paint
2. **Network Activity**: Request count, transfer sizes
3. **JavaScript Performance**: Bundle sizes, execution times
4. **Core Web Vitals**: LCP, FID, CLS measurements
```

## Troubleshooting

### Common Issues

**Domain Access Denied**
```
Error: net::ERR_BLOCKED_BY_CLIENT
```
- **Solution**: Add required domains to `allowed_domains` configuration
- **Check**: Verify domain spelling and wildcard patterns

**Container Startup Failed**
```
Error: Failed to start Playwright container
```
- **Solution**: Check Docker image version availability
- **Alternative**: Use `"latest"` for the most recent stable version

**Network Timeouts**
```
Error: Timeout exceeded while loading page
```
- **Solution**: Increase timeout values or check domain accessibility
- **Debug**: Verify the target website is responsive

### Performance Optimization

1. **Reuse browser contexts** for multiple page operations
2. **Disable unnecessary features** like images or CSS for data extraction
3. **Use headless mode** for faster execution in automation scenarios
4. **Implement retry logic** for network-dependent operations

### Debugging Tips

1. **Enable verbose logging** to see detailed browser operations
2. **Capture screenshots** at each step for visual debugging
3. **Monitor network activity** to identify failed requests
4. **Use browser developer tools** for interactive debugging

## Integration with Safe Outputs

Playwright workflows commonly use safe outputs to create issues, PRs, or comments based on findings:

```yaml
safe-outputs:
  create-issue:
    title-prefix: "[Playwright] "
    labels: [automation, testing]
    max: 5
    
  create-pull-request:
    title-prefix: "[Automated Fix] "
    branch-prefix: "playwright-fixes/"
    max: 2
```

## Best Practices

1. **Start with minimal domains**: Begin with `localhost` only and add domains as needed
2. **Use ecosystem bundles**: Prefer trusted domain bundles over individual domains
3. **Implement error handling**: Add retry logic for network-dependent operations
4. **Document domain requirements**: Clearly explain why specific domains are needed
5. **Regular maintenance**: Review and update domain allowlists periodically
6. **Test thoroughly**: Validate workflows in different network environments

## Related Documentation

- [Tools Configuration](tools.md) - General tool configuration guide
- [Safe Outputs](safe-outputs.md) - Automated issue and PR creation
- [Network Configuration](frontmatter.md#network) - Workflow-level network settings
- [Security Notes](security-notes.md) - Security best practices

## Example Applications

- **Web scraping and data extraction**
- **Automated testing and quality assurance**
- **Accessibility compliance auditing**
- **Performance monitoring and optimization**
- **Visual regression testing**
- **User interface automation**
- **Cross-browser compatibility testing**