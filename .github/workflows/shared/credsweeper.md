---
# CredSweeper Security Scanner
# Shared configuration for scanning credentials in workflow temporary files
#
# Usage:
#   imports:
#     - shared/credsweeper.md
#
# This import provides:
# - Automatic CredSweeper installation via post-steps
# - Scanning of /tmp/gh-aw/ directory for credentials
# - Masking of detected credentials using GitHub Actions secret masking
#
# Note: CredSweeper scans all files in /tmp/gh-aw/ after the AI agent completes.
# Any detected credentials are automatically masked using core.setSecret() to prevent
# exposure in logs or artifacts.

post-steps:
  - name: Install CredSweeper
    id: install-credsweeper
    run: |
      python -m pip install --upgrade pip
      pip install credsweeper
      credsweeper --version
  
  - name: Run CredSweeper
    id: run-credsweeper
    run: |
      # Create output directory
      mkdir -p /tmp/gh-aw/credsweeper
      
      # Run CredSweeper on /tmp/gh-aw/ directory
      echo "🔍 Scanning /tmp/gh-aw/ for credentials..."
      python -m credsweeper --path /tmp/gh-aw/ --save-json /tmp/gh-aw/credsweeper/output.json || true
      
      # Display results if file exists
      if [ -f /tmp/gh-aw/credsweeper/output.json ]; then
        echo "📄 CredSweeper scan complete. Results saved to /tmp/gh-aw/credsweeper/output.json"
        
        # Show summary
        CREDENTIAL_COUNT=$(jq 'length' /tmp/gh-aw/credsweeper/output.json 2>/dev/null || echo "0")
        echo "Found $CREDENTIAL_COUNT potential credential(s)"
        
        if [ "$CREDENTIAL_COUNT" != "0" ]; then
          echo "⚠️  WARNING: Credentials detected in temporary files!"
          echo "These will be masked in the next step to prevent exposure."
        fi
      else
        echo "✅ No credentials detected"
        echo "[]" > /tmp/gh-aw/credsweeper/output.json
      fi
  
  - name: Mask CredSweeper Findings
    if: always()
    uses: actions/github-script@v8
    with:
      script: |
        const fs = require('fs');
        const path = require('path');
        
        /**
         * Process CredSweeper output and mask all detected credentials
         */
        async function maskCredentials() {
          const outputPath = '/tmp/gh-aw/credsweeper/output.json';
          
          // Check if output file exists
          if (!fs.existsSync(outputPath)) {
            core.info('No CredSweeper output file found, skipping masking');
            return;
          }
          
          // Read and parse CredSweeper output
          let credentials;
          try {
            const content = fs.readFileSync(outputPath, 'utf8');
            credentials = JSON.parse(content);
          } catch (error) {
            core.warning(`Failed to parse CredSweeper output: ${error instanceof Error ? error.message : String(error)}`);
            return;
          }
          
          if (!Array.isArray(credentials) || credentials.length === 0) {
            core.info('✅ No credentials detected by CredSweeper');
            return;
          }
          
          core.info(`🔒 Processing ${credentials.length} potential credential(s) for masking`);
          
          let maskedCount = 0;
          
          // Process each credential finding
          for (const credential of credentials) {
            // Skip if no line_data_list
            if (!credential.line_data_list || !Array.isArray(credential.line_data_list)) {
              continue;
            }
            
            // Process each line data entry
            for (const lineData of credential.line_data_list) {
              // Get the credential value
              const value = lineData.value;
              
              // Skip empty or very short values (likely false positives)
              if (!value || value.length < 8) {
                continue;
              }
              
              // Mask the credential using GitHub Actions secret masking
              core.setSecret(value);
              maskedCount++;
              
              // Log details (the value itself will be masked in logs)
              const rule = credential.rule || 'Unknown';
              const filePath = lineData.path || 'Unknown';
              const lineNum = lineData.line_num || 'N/A';
              
              core.warning(
                `Masked credential: ${rule} in ${filePath}:${lineNum} ` +
                `(confidence: ${credential.confidence || 'N/A'}, severity: ${credential.severity || 'N/A'})`
              );
            }
          }
          
          if (maskedCount > 0) {
            core.warning(`⚠️  Masked ${maskedCount} credential value(s) to prevent exposure in logs`);
            core.summary.addHeading('CredSweeper Security Scan', 2);
            core.summary.addRaw(`⚠️  Detected and masked ${maskedCount} potential credential(s)\n\n`);
            core.summary.addRaw('These credentials have been automatically masked using GitHub Actions secret masking to prevent exposure in workflow logs and artifacts.\n\n');
            core.summary.addRaw(`Total findings: ${credentials.length}\n`);
            await core.summary.write();
          } else {
            core.info('✅ No credentials required masking');
          }
        }
        
        await maskCredentials();
---

# CredSweeper Security Scanner

CredSweeper is installed and will automatically scan the `/tmp/gh-aw/` directory after your workflow completes.

## What It Does

1. **Scans for Credentials**: CredSweeper scans all files in `/tmp/gh-aw/` for potential credentials such as:
   - API keys
   - Passwords
   - Tokens
   - Private keys
   - Database connection strings
   - And other sensitive information

2. **Automatic Masking**: Any detected credentials are automatically masked using GitHub Actions `core.setSecret()` to prevent them from appearing in:
   - Workflow logs
   - Step outputs
   - Artifacts
   - Job summaries

3. **Security Report**: A summary of detected credentials is added to the job summary, showing:
   - Number of credentials found
   - Type of credential (rule name)
   - File location
   - Confidence and severity levels

## How It Works

- **Installation**: Python and CredSweeper are installed via pip
- **Scanning**: CredSweeper scans `/tmp/gh-aw/` directory recursively
- **Output**: Results are saved to `/tmp/gh-aw/credsweeper/output.json`
- **Masking**: JavaScript processes the JSON output and masks each detected value

## Security Benefits

This shared workflow helps prevent accidental exposure of credentials that may be:
- Generated by the AI agent
- Stored in temporary files
- Included in logs or debug output
- Present in downloaded content or artifacts

By automatically scanning and masking credentials, this workflow adds an extra layer of security to your agentic workflows.

## Requirements

- Python 3.10+ (pre-installed on GitHub Actions runners)
- Internet access to install CredSweeper from PyPI

## Configuration

No additional configuration is needed. Simply import this shared workflow:

```yaml
---
imports:
  - shared/credsweeper.md
---
```

The post-steps will automatically run after your AI agent completes, regardless of whether the agent succeeds or fails.
