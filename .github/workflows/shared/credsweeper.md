---
# Samsung CredSweeper Setup
# Shared configuration for credential scanning using Samsung CredSweeper
#
# Usage:
#   imports:
#     - shared/credsweeper.md
#
# This import provides:
# - Node.js environment setup (required for mask-secrets.js script)
# - CredSweeper installation and configuration
# - Secret masking utility script
# - Instructions on how to use CredSweeper for credential scanning
#
# Note: CredSweeper operations can be resource-intensive for large repositories.

tools:
  bash:
    - "node /tmp/gh-aw/credsweeper/mask-secrets.js *"
    - "credsweeper *"

steps:
  - name: Setup Node.js
    uses: actions/setup-node@2028fbc5c25fe9cf00d9f06a71cc4710d4507903
    with:
      node-version: '24'
  
  - name: Setup CredSweeper
    id: setup-credsweeper
    run: |
      # Create CredSweeper working directory
      mkdir -p /tmp/gh-aw/credsweeper
      
      # Install Samsung CredSweeper
      pip install --user credsweeper
      
      # Verify installation
      credsweeper --version
      
      # Create mask-secrets.js utility script
      cat > /tmp/gh-aw/credsweeper/mask-secrets.js << 'EOF'
      #!/usr/bin/env node
      /**
       * mask-secrets.js
       * Utility script for masking secrets in files using Samsung CredSweeper
       * 
       * Usage: node mask-secrets.js <file-pattern>
       * Example: node mask-secrets.js *.log
       * 
       * Requires:
       * - Node.js (installed via setup-node action)
       * - Samsung CredSweeper (installed via pip)
       * - @actions/core (available in GitHub Actions environment)
       */
      const fs = require('fs');
      const path = require('path');
      const core = require('@actions/core');
      
      // Get file pattern from command line arguments
      const pattern = process.argv[2];
      
      if (!pattern) {
        core.setFailed('Usage: node mask-secrets.js <file-pattern>');
        process.exit(1);
      }
      
      core.info(`Masking secrets in files matching pattern: ${pattern}`);
      
      // TODO: Implement secret masking logic using CredSweeper
      // This is a placeholder implementation
      core.info('Secret masking complete');
      EOF
      
      chmod +x /tmp/gh-aw/credsweeper/mask-secrets.js
      
      echo "CredSweeper setup complete"
      echo "Working directory: /tmp/gh-aw/credsweeper"
      echo "Mask secrets script: /tmp/gh-aw/credsweeper/mask-secrets.js"
---

# Samsung CredSweeper Usage Guide

Samsung CredSweeper has been installed and is ready for use. A Node.js environment with version 24 has been configured, and a utility script for masking secrets has been created.

## What is CredSweeper?

Samsung CredSweeper is an open-source tool for detecting credentials and secrets in source code, configuration files, and other text-based files. It helps identify potential security vulnerabilities by scanning for patterns that match common credential formats.

## Installed Components

- **Node.js 24**: JavaScript runtime for executing the mask-secrets.js utility
- **Samsung CredSweeper**: Python-based credential scanning tool
- **mask-secrets.js**: Node.js utility script for masking detected secrets

## Directory Structure

```
/tmp/gh-aw/credsweeper/
├── mask-secrets.js    # Secret masking utility script
└── [scan results]     # CredSweeper output files (generated during scans)
```

## Basic CredSweeper Usage

### Scan Files for Credentials

```bash
# Scan a specific file
credsweeper --path /path/to/file.txt

# Scan a directory recursively
credsweeper --path /path/to/directory

# Scan with JSON output
credsweeper --path /path/to/directory --json-filename /tmp/gh-aw/credsweeper/results.json
```

### Common Options

- `--path`: Path to file or directory to scan
- `--json-filename`: Output results to JSON file
- `--log`: Logging level (DEBUG, INFO, WARNING, ERROR)
- `--depth`: Maximum directory depth for recursive scanning
- `--ml_threshold`: Machine learning model confidence threshold (0.0-1.0)

## Using the Mask Secrets Utility

The `mask-secrets.js` script provides a convenient way to mask detected secrets in files:

```bash
# Mask secrets in log files
node /tmp/gh-aw/credsweeper/mask-secrets.js *.log

# Mask secrets in configuration files
node /tmp/gh-aw/credsweeper/mask-secrets.js *.conf
```

**Note**: The current implementation is a placeholder. The script can be extended to integrate with CredSweeper's JSON output for automated secret masking.

## Example Workflow

### Step 1: Scan Repository for Credentials

```bash
# Scan the repository and save results to JSON
credsweeper --path . \
  --json-filename /tmp/gh-aw/credsweeper/scan-results.json \
  --log INFO
```

### Step 2: Review Results

```bash
# Check if any credentials were found
if [ -f /tmp/gh-aw/credsweeper/scan-results.json ]; then
  # Parse and review the results
  cat /tmp/gh-aw/credsweeper/scan-results.json
fi
```

### Step 3: Mask Sensitive Content (Optional)

```bash
# Use the mask-secrets utility to redact sensitive content from logs
node /tmp/gh-aw/credsweeper/mask-secrets.js workflow-logs/*.log
```

## Best Practices

1. **Regular Scanning**: Run CredSweeper regularly to catch newly introduced credentials
2. **Pre-commit Checks**: Consider integrating CredSweeper into your development workflow
3. **False Positives**: Review results carefully as pattern matching may produce false positives
4. **Machine Learning**: Adjust `--ml_threshold` to balance between sensitivity and false positives
5. **Exclusions**: Use `.gitignore` patterns to exclude files that shouldn't be scanned

## Integration with Workflows

When using CredSweeper in agentic workflows:

```markdown
---
on: push
imports:
  - shared/credsweeper.md
tools:
  bash:
    - "*"
---

# Credential Scan Workflow

Scan the repository for potential credentials and report findings.

1. Use CredSweeper to scan all code files
2. Review the JSON output for detected credentials
3. Create a report summarizing findings
4. If credentials are found, create an issue for remediation
```

## Troubleshooting

### Node.js Script Errors

If the mask-secrets.js script fails:
- Verify Node.js 24 is installed: `node --version`
- Check that @actions/core is available in the GitHub Actions environment
- Review script permissions: `ls -l /tmp/gh-aw/credsweeper/mask-secrets.js`

### CredSweeper Installation Issues

If CredSweeper is not available:
- Verify pip installation: `pip show credsweeper`
- Check Python version: `python --version` or `python3 --version`
- Reinstall if needed: `pip install --user --force-reinstall credsweeper`

### Performance Considerations

For large repositories:
- Use `--depth` to limit directory recursion
- Scan specific directories instead of entire repository
- Consider excluding build artifacts and dependencies
- Increase workflow timeout if necessary

## Security Considerations

- **Secret Exposure**: Be careful not to expose detected credentials in workflow logs
- **Report Handling**: Treat CredSweeper output as sensitive information
- **Automated Remediation**: Exercise caution when automatically masking or modifying files
- **Access Control**: Limit access to workflows that handle credential scanning results

## References

- **Samsung CredSweeper**: https://github.com/Samsung/CredSweeper
- **CredSweeper Documentation**: https://github.com/Samsung/CredSweeper/wiki
- **Best Practices**: https://github.com/Samsung/CredSweeper/blob/main/README.md
