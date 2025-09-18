---
title: Custom Engine
description: Complete guide to the Custom engine for executing user-defined GitHub Actions steps without AI interpretation.
sidebar:
  order: 3
---

The Custom engine allows you to define traditional GitHub Actions steps that execute without AI interpretation. This is useful for deterministic workflows, hybrid AI/traditional approaches, or when you need precise control over workflow execution.

## Basic Configuration

### Simple Configuration

```yaml
engine: custom
```

### Extended Configuration

```yaml
engine:
  id: custom
  max-turns: 3    # Optional: for consistency (no effect)
  env:            # Optional: environment variables
    NODE_ENV: production
    DEBUG: "true"
  steps:          # Required: GitHub Actions steps
    - name: Setup Node.js
      uses: actions/setup-node@v4
      with:
        node-version: '18'
    - name: Install dependencies
      run: npm install
    - name: Run tests
      run: npm test
      env:
        CI: true
```

## Frontmatter Configuration

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `id` | string | `custom` | Engine identifier (required) |
| `max-turns` | number | none | Kept for consistency (no effect) |
| `env` | object | none | Environment variables for all steps |
| `steps` | array | none | GitHub Actions steps to execute |

### Steps Configuration

The `steps` field contains an array of standard GitHub Actions step definitions:

```yaml
engine:
  id: custom
  steps:
    # Action-based step
    - name: Checkout code
      uses: actions/checkout@v4
      with:
        fetch-depth: 1
    
    # Command-based step
    - name: Run custom script
      run: |
        echo "Starting custom workflow"
        ./scripts/custom-logic.sh
      working-directory: ./src
    
    # Step with conditional execution
    - name: Deploy to staging
      if: github.ref == 'refs/heads/develop'
      run: ./deploy.sh staging
      env:
        DEPLOY_KEY: ${{ secrets.STAGING_DEPLOY_KEY }}
    
    # Multi-line command step
    - name: Complex build process
      run: |
        set -e
        echo "Building application..."
        make clean
        make build
        make test
        echo "Build completed successfully"
```

### Environment Variables

Environment variables can be set at the engine level and will be merged with step-specific environment variables:

```yaml
engine:
  id: custom
  env:
    # Global environment variables for all steps
    NODE_ENV: production
    API_BASE_URL: https://api.example.com
    DEBUG_MODE: "true"
  steps:
    - name: Run application
      run: node app.js
      env:
        # Step-specific environment variables
        PORT: 3000
        # Inherits NODE_ENV, API_BASE_URL, DEBUG_MODE from engine level
```

## Version Control

The Custom engine does not have version control in the traditional sense since it executes user-defined GitHub Actions steps directly. However, you can control versions of the actions you use:

### Action Version Specification

```yaml
engine:
  id: custom
  steps:
    # Pin to specific major version
    - name: Setup Node.js
      uses: actions/setup-node@v4
      
    # Pin to specific minor version
    - name: Cache dependencies
      uses: actions/cache@v3.3
      
    # Pin to specific commit SHA (most secure)
    - name: Deploy to AWS
      uses: aws-actions/configure-aws-credentials@1e326a4557363cd93a3e77a6c0e2c6daf9c6b537
```

### Best Practices for Versions

1. **Pin Major Versions**: Use `@v4` instead of `@latest`
2. **Security Critical**: Use SHA pins for security-sensitive actions
3. **Regular Updates**: Keep action versions up to date
4. **Documentation**: Document why specific versions are chosen

## Network Isolation

The Custom engine does not implement network isolation like the AI engines. Network access depends on the GitHub Actions runner environment and the steps you define.

### How Network Access Works

1. **Runner Environment**: Network access is controlled by GitHub Actions runner
2. **Step-level Control**: Individual steps can implement their own network restrictions
3. **No Global Hooks**: No system-wide network interception
4. **Manual Implementation**: Network restrictions must be implemented in your steps

### Implementing Network Restrictions

You can implement network restrictions in your custom steps:

```yaml
engine:
  id: custom
  steps:
    # Setup network restrictions using iptables (Linux runners)
    - name: Configure network restrictions
      run: |
        # Block all outbound except specific domains
        sudo iptables -A OUTPUT -d api.github.com -j ACCEPT
        sudo iptables -A OUTPUT -d *.example.com -j ACCEPT
        sudo iptables -A OUTPUT -j DROP
    
    # Run your restricted application
    - name: Run application with network restrictions
      run: ./my-application
```

### Proxy Configuration

For more sophisticated network control, you can use proxy servers:

```yaml
engine:
  id: custom
  steps:
    - name: Setup proxy
      run: |
        # Configure HTTP proxy for network filtering
        export HTTP_PROXY=http://filtering-proxy:8080
        export HTTPS_PROXY=http://filtering-proxy:8080
        export NO_PROXY=localhost,127.0.0.1
    
    - name: Run with proxy
      run: curl https://api.example.com/data
      env:
        HTTP_PROXY: http://filtering-proxy:8080
        HTTPS_PROXY: http://filtering-proxy:8080
```

## Features

### Core Capabilities

- **Direct Execution**: No AI interpretation, runs exactly as written
- **Standard Actions**: Full compatibility with GitHub Actions ecosystem
- **Conditional Logic**: Support for `if` conditions and expressions
- **Environment Control**: Flexible environment variable management

### Supported Features

| Feature | Supported | Description |
|---------|-----------|-------------|
| Max Turns | ⚠️ | Accepted but has no effect |
| Tools Whitelist | ❌ | Not applicable to custom steps |
| HTTP Transport | ❌ | Not applicable to custom steps |
| Network Isolation | ❌ | Must be implemented manually |
| Custom Environment | ✅ | Full environment variable support |
| Version Control | ✅ | Through action version pinning |
| Conditional Execution | ✅ | Full GitHub Actions `if` support |
| Matrix Builds | ✅ | Can be used in matrix strategies |

### GitHub Actions Integration

The Custom engine generates standard GitHub Actions steps:

```yaml
# Generated workflow step
- name: Setup Node.js
  uses: actions/setup-node@v4
  with:
    node-version: '18'

- name: Install dependencies
  run: npm install
  env:
    NODE_ENV: production
    DEBUG_MODE: "true"
    GITHUB_AW_PROMPT: /tmp/aw-prompts/prompt.txt

- name: Ensure log file exists
  run: |
    echo "Custom steps execution completed" >> /tmp/aw-logs/log.txt
    touch /tmp/aw-logs/log.txt
```

### Automatic Environment Variables

The Custom engine automatically adds:

- `GITHUB_AW_PROMPT`: Path to the prompt file (`/tmp/aw-prompts/prompt.txt`)
- User-defined environment variables from the `env` section
- Step-specific environment variables

## Execution Process

### No Installation Phase

The Custom engine has no installation requirements since it uses standard GitHub Actions steps.

### Execution Phase

1. **Step Conversion**: Convert step definitions to YAML
2. **Environment Merging**: Merge engine-level and step-level environment variables
3. **Direct Execution**: Execute steps as standard GitHub Actions
4. **Log Generation**: Create log file for consistency

### Generated Steps

For each custom step, the engine:

1. **Preserves Structure**: Maintains exact step definition
2. **Merges Environment**: Combines engine and step environment variables
3. **Adds Prompt**: Automatically includes `GITHUB_AW_PROMPT` environment variable

## Use Cases

### Hybrid Workflows

Combine AI engines with custom steps:

```yaml
# File: .github/workflows/hybrid-workflow.md
---
on: push
engine: claude  # AI engine for main logic
---

# AI Prompt
Analyze the code changes and generate a summary.

<!-- Use include directive for custom steps -->
<!-- include: .github/workflows/custom-deploy.md -->
```

```yaml
# File: .github/workflows/custom-deploy.md
---
engine: custom  # Custom steps for deployment
---

```yaml
engine:
  id: custom
  steps:
    - name: Deploy to production
      run: ./deploy.sh
      if: github.ref == 'refs/heads/main'
```

### Testing and Development

Use custom engine for testing workflow components:

```yaml
engine:
  id: custom
  steps:
    - name: Test step execution
      run: echo "Testing workflow step"
    
    - name: Validate environment
      run: |
        echo "Node version: $(node --version)"
        echo "NPM version: $(npm --version)"
        echo "Current directory: $(pwd)"
        echo "Environment: $NODE_ENV"
```

### Legacy Workflow Migration

Migrate existing GitHub Actions workflows:

```yaml
# Original .github/workflows/ci.yml
# jobs:
#   test:
#     steps:
#       - uses: actions/checkout@v4
#       - uses: actions/setup-node@v4
#       - run: npm test

# Migrated to gh-aw custom engine
engine:
  id: custom
  steps:
    - name: Checkout code
      uses: actions/checkout@v4
    - name: Setup Node.js
      uses: actions/setup-node@v4
      with:
        node-version: '18'
    - name: Run tests
      run: npm test
```

## Security Considerations

### No Built-in Isolation

The Custom engine provides no built-in security isolation:

- **Network Access**: Full network access by default
- **File System**: Full file system access
- **Secrets**: Access to all repository secrets
- **Permissions**: Inherits workflow permissions

### Manual Security Implementation

Implement security measures in your steps:

```yaml
engine:
  id: custom
  steps:
    # Validate inputs
    - name: Validate inputs
      run: |
        if [[ ! "$INPUT_VALUE" =~ ^[a-zA-Z0-9_-]+$ ]]; then
          echo "Invalid input format"
          exit 1
        fi
    
    # Run with restricted permissions
    - name: Run with limited user
      run: |
        # Create limited user
        sudo useradd -m -s /bin/bash limited_user
        # Run command as limited user
        sudo -u limited_user ./safe-command.sh
    
    # Clean up sensitive data
    - name: Cleanup
      if: always()
      run: |
        rm -f /tmp/sensitive-data.txt
        unset SECRET_VALUE
```

### Secret Management

Handle secrets carefully in custom steps:

```yaml
engine:
  id: custom
  steps:
    - name: Use secrets securely
      run: |
        # Don't echo secrets
        # echo "$SECRET_KEY"  # DON'T DO THIS
        
        # Use secrets in variables without logging
        export API_KEY="$SECRET_KEY"
        ./app --api-key-from-env
      env:
        SECRET_KEY: ${{ secrets.API_SECRET }}
```

## Troubleshooting

### Common Issues

**Step YAML Syntax Error**
```
Error: Invalid step definition in custom engine
```
Solution: Validate YAML syntax in step definitions.

**Environment Variable Not Available**
```
Error: Required environment variable not set
```
Solution: Check that variables are defined in engine `env` or step `env`.

**Action Version Not Found**
```
Error: Unable to resolve action
```
Solution: Verify action exists and version is correct.

**Permission Denied**
```
Error: Permission denied running step
```
Solution: Check file permissions or use `sudo` if needed.

### Debugging Custom Steps

Add debugging information to steps:

```yaml
engine:
  id: custom
  env:
    DEBUG: "true"
  steps:
    - name: Debug environment
      run: |
        echo "=== Environment Debug ==="
        echo "Current user: $(whoami)"
        echo "Current directory: $(pwd)"
        echo "Environment variables:"
        env | sort
        echo "=== End Debug ==="
    
    - name: Your actual step
      run: ./your-command
```

### Step Validation

Validate step definitions before deployment:

```bash
# Use GitHub Actions CLI to validate
gh workflow view .github/workflows/your-workflow.yml

# Or use act to test locally
act -n  # Dry run to check syntax
```

## Migration to Custom Engine

### From AI Engines

When migrating from AI engines to custom steps:

1. **Define Explicit Steps**: Convert AI instructions to specific actions
2. **Handle Environment**: Move environment setup to custom steps
3. **Implement Security**: Add manual security measures
4. **Remove AI Features**: Remove AI-specific configuration

```yaml
# AI engine (Claude)
engine: claude
tools:
  - bash
  - github
  
# Equivalent custom engine
engine:
  id: custom
  steps:
    - name: Setup environment
      run: |
        # Setup equivalent to AI tools
        apt-get update && apt-get install -y curl jq
    
    - name: Execute logic
      run: |
        # Explicit commands instead of AI interpretation
        ./specific-script.sh
        curl -H "Authorization: token $GITHUB_TOKEN" \
             https://api.github.com/repos/owner/repo/issues
```

## Related Documentation

- [GitHub Actions Documentation](https://docs.github.com/en/actions) - Complete GitHub Actions reference
- [Workflow Syntax](https://docs.github.com/en/actions/using-workflows/workflow-syntax-for-github-actions) - GitHub Actions YAML syntax
- [Include Directives](/gh-aw/reference/include-directives/) - Combining multiple workflow files
- [Hybrid Workflows](/gh-aw/guides/hybrid-workflows/) - Combining AI and custom engines

## External Links

- [GitHub Actions Marketplace](https://github.com/marketplace?type=actions) - Browse available actions
- [GitHub Actions Toolkit](https://github.com/actions/toolkit) - Build custom actions
- [act](https://github.com/nektos/act) - Run GitHub Actions locally
- [Super-Linter](https://github.com/github/super-linter) - Validate workflow files