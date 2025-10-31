---
# Samsung CredSweeper Setup
# Shared configuration for using Samsung CredSweeper credential scanner in workflows
#
# Usage:
#   imports:
#     - shared/credsweeper.md
#
# This import provides:
# - Automatic Docker pull of Samsung/credsweeper image
# - Instructions on how to use credsweeper for scanning
# - Best practices for credential detection
#
# Note: CredSweeper scans can be time-intensive for large codebases.
# Ensure your workflow has adequate timeout_minutes (recommended: 10+ minutes).

tools:
  bash:
    - "docker run *"
    - "docker pull *"
    - "docker ps *"
    - "docker images *"
    - "node /tmp/gh-aw/credsweeper/mask-secrets.js *"

steps:
  - name: Setup CredSweeper
    id: setup-credsweeper
    run: |
      echo "🔍 Pulling Samsung CredSweeper Docker image..."
      docker pull ghcr.io/samsung/credsweeper:latest
      echo "✅ CredSweeper image pulled successfully"
      docker images ghcr.io/samsung/credsweeper:latest
      
      # Create temporary directory for scans
      mkdir -p /tmp/gh-aw/credsweeper
      echo "📁 Created /tmp/gh-aw/credsweeper for scan results"
      
      # Create JavaScript script to parse and mask secrets
      cat > /tmp/gh-aw/credsweeper/mask-secrets.js << 'EOF'
      #!/usr/bin/env node
      // mask-secrets.js
      // Parse CredSweeper JSON results and mask secrets with ***
      
      const fs = require('fs');
      
      function maskSecret(secret) {
        if (!secret || secret.length === 0) return '***';
        if (secret.length <= 6) return '***';
        // Show first 2 and last 2 characters, mask the rest
        return secret.substring(0, 2) + '***' + secret.substring(secret.length - 2);
      }
      
      function processResults(inputPath) {
        try {
          const data = fs.readFileSync(inputPath, 'utf8');
          const results = JSON.parse(data);
          
          if (!Array.isArray(results)) {
            console.error('Error: Expected JSON array');
            process.exit(1);
          }
          
          // Process each finding and mask the secrets
          const maskedResults = results.map(finding => {
            const maskedFinding = { ...finding };
            
            // Mask line_data_list entries
            if (maskedFinding.line_data_list && Array.isArray(maskedFinding.line_data_list)) {
              maskedFinding.line_data_list = maskedFinding.line_data_list.map(lineData => {
                const maskedLineData = { ...lineData };
                
                // Mask the actual secret in the line
                if (maskedLineData.line && maskedLineData.value) {
                  maskedLineData.line = maskedLineData.line.replace(
                    maskedLineData.value,
                    maskSecret(maskedLineData.value)
                  );
                }
                
                // Mask the value field
                if (maskedLineData.value) {
                  maskedLineData.value = maskSecret(maskedLineData.value);
                }
                
                return maskedLineData;
              });
            }
            
            return maskedFinding;
          });
          
          // Output masked results as JSON
          console.log(JSON.stringify(maskedResults, null, 2));
          
        } catch (error) {
          console.error('Error processing file:', error.message);
          process.exit(1);
        }
      }
      
      // Main execution
      const args = process.argv.slice(2);
      if (args.length === 0) {
        console.error('Usage: node mask-secrets.js <credsweeper-results.json>');
        process.exit(1);
      }
      
      processResults(args[0]);
      EOF
      chmod +x /tmp/gh-aw/credsweeper/mask-secrets.js
---

<!--
# Samsung CredSweeper Usage Guide

Samsung CredSweeper has been set up and is ready to use. The Docker image `ghcr.io/samsung/credsweeper:latest` is available, and a temporary folder `/tmp/gh-aw/credsweeper` is ready for scan results.

A JavaScript utility script is available at `/tmp/gh-aw/credsweeper/mask-secrets.js` to parse CredSweeper results and mask secrets with `***` for safe display.

**Note**: CredSweeper scans can take several minutes for large codebases. Individual bash commands have a 5-minute timeout by default. For longer scans, increase workflow timeout_minutes.

## About CredSweeper

CredSweeper is a tool to detect credentials (API keys, tokens, passwords, etc.) in:
- Source code files
- Configuration files  
- Git repositories
- Text documents

It uses machine learning and pattern matching to identify various types of credentials while minimizing false positives.

## Basic Usage

### Scan Files in /tmp/gh-aw/

The most common use case is to scan files that have been downloaded or created in the `/tmp/gh-aw/` directory:

```bash
# Scan all files in /tmp/gh-aw/ directory
docker run --rm -v /tmp/gh-aw:/code ghcr.io/samsung/credsweeper:latest --path /code --save-json /code/credsweeper/scan-results.json

# Scan with output to console
docker run --rm -v /tmp/gh-aw:/code ghcr.io/samsung/credsweeper:latest --path /code
```

**Key flags:**
- `--rm`: Remove container after scan completes
- `-v /tmp/gh-aw:/code`: Mount the /tmp/gh-aw directory as /code in the container
- `--path /code`: Directory to scan inside the container
- `--save-json <path>`: Save results as JSON file
- `--log <level>`: Set log level (critical, error, warning, info, debug)

### Scan Specific Files

```bash
# Scan a specific file
docker run --rm -v /tmp/gh-aw:/code ghcr.io/samsung/credsweeper:latest \
  --path /code/myfile.py \
  --save-json /code/credsweeper/results.json

# Scan multiple specific files
docker run --rm -v /tmp/gh-aw:/code ghcr.io/samsung/credsweeper:latest \
  --path /code/file1.js \
  --path /code/file2.py \
  --save-json /code/credsweeper/results.json
```

### Advanced Options

```bash
# Scan with ML validation (more accurate but slower)
docker run --rm -v /tmp/gh-aw:/code ghcr.io/samsung/credsweeper:latest \
  --path /code \
  --ml_validation \
  --save-json /code/credsweeper/results.json

# Include debug information
docker run --rm -v /tmp/gh-aw:/code ghcr.io/samsung/credsweeper:latest \
  --path /code \
  --log debug \
  --save-json /code/credsweeper/results.json

# Skip specific credential types
docker run --rm -v /tmp/gh-aw:/code ghcr.io/samsung/credsweeper:latest \
  --path /code \
  --skip-ignored \
  --save-json /code/credsweeper/results.json
```

**Advanced flags:**
- `--ml_validation`: Use machine learning for validation (slower but more accurate)
- `--skip-ignored`: Skip credentials in ignore lists
- `--depth <n>`: Maximum depth for directory scanning
- `--jobs <n>`: Number of parallel jobs (default: CPU count)
- `--banner`: Show banner with ASCII art
- `--api_validation`: Validate credentials against live APIs (requires network access)

## Output Formats

### JSON Output

The scan results are saved as JSON with the following structure:

```json
[
  {
    "rule": "Password",
    "severity": "high",
    "line_data_list": [
      {
        "line": "password = 'my_secret_password'",
        "line_num": 42,
        "path": "config.py",
        "info": "Password in plain text"
      }
    ],
    "ml_validation": "VALIDATED_KEY"
  }
]
```

### Reading Results

```bash
# Parse and mask secrets for safe display
node /tmp/gh-aw/credsweeper/mask-secrets.js /tmp/gh-aw/credsweeper/scan-results.json

# Save masked results to a file
node /tmp/gh-aw/credsweeper/mask-secrets.js /tmp/gh-aw/credsweeper/scan-results.json > /tmp/gh-aw/credsweeper/masked-results.json

# Pretty-print JSON results (unmasked - be careful with sensitive data)
cat /tmp/gh-aw/credsweeper/scan-results.json | jq '.'

# Count findings
cat /tmp/gh-aw/credsweeper/scan-results.json | jq 'length'

# List unique credential types found
cat /tmp/gh-aw/credsweeper/scan-results.json | jq '.[].rule' | sort | uniq

# Filter high severity findings
cat /tmp/gh-aw/credsweeper/scan-results.json | jq '.[] | select(.severity == "high")'

# Use masked results with jq
node /tmp/gh-aw/credsweeper/mask-secrets.js /tmp/gh-aw/credsweeper/scan-results.json | jq '.[] | select(.severity == "high")'
```

## Common Workflows

### Scan and Report Findings

```bash
# Run scan
docker run --rm -v /tmp/gh-aw:/code ghcr.io/samsung/credsweeper:latest \
  --path /code \
  --save-json /code/credsweeper/results.json

# Check if any credentials were found and display masked results
FINDINGS=$(cat /tmp/gh-aw/credsweeper/results.json | jq 'length')
if [ "$FINDINGS" -gt 0 ]; then
  echo "⚠️ Found $FINDINGS potential credentials"
  # Display masked results for safe output
  node /tmp/gh-aw/credsweeper/mask-secrets.js /tmp/gh-aw/credsweeper/results.json | jq '.[]'
else
  echo "✅ No credentials found"
fi
```

### Scan with Summary Statistics

```bash
# Run scan and save results
docker run --rm -v /tmp/gh-aw:/code ghcr.io/samsung/credsweeper:latest \
  --path /code \
  --save-json /code/credsweeper/results.json

# Generate summary
echo "### CredSweeper Scan Summary"
echo "Total findings: $(cat /tmp/gh-aw/credsweeper/results.json | jq 'length')"
echo ""
echo "Findings by type:"
cat /tmp/gh-aw/credsweeper/results.json | jq -r '.[].rule' | sort | uniq -c
echo ""
echo "Findings by severity:"
cat /tmp/gh-aw/credsweeper/results.json | jq -r '.[].severity' | sort | uniq -c
```

## Best Practices

1. **Target Specific Directories**: Scan only relevant directories to reduce scan time
2. **Use ML Validation**: Enable `--ml_validation` for production scans to reduce false positives
3. **Review Results**: Always review findings manually as automated tools can have false positives
4. **Cache Docker Image**: The setup step pulls the image once; subsequent runs will use the cached image
5. **Save Results**: Always use `--save-json` to persist results for later analysis
6. **Handle Large Codebases**: For very large codebases, consider increasing timeout or scanning incrementally

## Security Considerations

- CredSweeper scans for credentials but does not validate them against live services by default
- Results may contain sensitive information; handle scan output files carefully
- Use `--api_validation` cautiously as it makes network requests to validate credentials
- Review all findings before taking action (rotating keys, updating secrets)

## Troubleshooting

### Docker Permission Issues

If you encounter permission errors:

```bash
# Ensure Docker daemon is running
docker ps

# Check Docker is accessible
docker run --rm hello-world
```

### Large Output Files

For large codebases with many findings:

```bash
# Filter results to only high severity
cat /tmp/gh-aw/credsweeper/results.json | jq '[.[] | select(.severity == "high")]' > /tmp/gh-aw/credsweeper/high-severity.json
```

### Scan Timeout

If scans timeout on large codebases:

1. Increase workflow `timeout_minutes`
2. Use `--jobs` to limit parallel processing
3. Scan subdirectories separately
-->
