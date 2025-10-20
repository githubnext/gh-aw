---
safe-outputs:
  jobs:
    mask-secrets:
      name: Mask Secrets with CredSweeper
      description: Scans agent output text and replaces detected secrets with (redacted)
      runs-on: ubuntu-latest
      permissions:
        contents: read
      inputs:
        text:
          description: "The text content to scan for secrets (defaults to agent output)"
          required: false
          type: string
          default: ""
      steps:
        - name: Install CredSweeper
          run: |
            pip install credsweeper
        
        - name: Scan and Mask Secrets
          run: |
            set -e
            
            # Read the agent output
            AGENT_OUTPUT_FILE="${GH_AW_AGENT_OUTPUT}"
            
            if [ ! -f "$AGENT_OUTPUT_FILE" ]; then
              echo "Error: Agent output file not found at $AGENT_OUTPUT_FILE"
              exit 1
            fi
            
            echo "Processing agent output from: $AGENT_OUTPUT_FILE"
            
            # Extract the text content from the agent output JSON  
            # This could be in .text field or in items with mask_secrets type
            OUTPUT_TEXT=$(jq -r '.text // ""' "$AGENT_OUTPUT_FILE")
            
            # Also check for mask-secrets items in the output
            MASK_SECRETS_TEXT=$(jq -r '[.items[]? | select(.type == "mask_secrets") | .text] | join("\n")' "$AGENT_OUTPUT_FILE")
            
            # Combine both sources of text
            if [ -n "$MASK_SECRETS_TEXT" ] && [ "$MASK_SECRETS_TEXT" != "" ]; then
              OUTPUT_TEXT="${OUTPUT_TEXT}${MASK_SECRETS_TEXT}"
            fi
            
            if [ -z "$OUTPUT_TEXT" ] || [ "$OUTPUT_TEXT" == "" ]; then
              echo "No text content found in agent output"
              exit 0
            fi
            
            echo "Found text to scan (${#OUTPUT_TEXT} characters)"
            
            # Save text to temporary file for scanning
            TEMP_FILE=$(mktemp)
            echo "$OUTPUT_TEXT" > "$TEMP_FILE"
            
            # Run CredSweeper to detect secrets
            echo "Running CredSweeper scan..."
            credsweeper --path "$TEMP_FILE" --json --log warning > /tmp/credsweeper-results.json || true
            
            # Process results and mask secrets
            MASKED_TEXT="$OUTPUT_TEXT"
            SECRET_COUNT=0
            
            # Extract all detected credentials and mask them
            if [ -f /tmp/credsweeper-results.json ] && [ -s /tmp/credsweeper-results.json ]; then
              echo "Processing CredSweeper results..."
              
              # Read each line and extract secrets
              while IFS= read -r line; do
                if [ -n "$line" ]; then
                  # Extract the secret value and line number
                  SECRET=$(echo "$line" | jq -r '.value // ""')
                  if [ -n "$SECRET" ] && [ "$SECRET" != "null" ] && [ "$SECRET" != "" ]; then
                    echo "Found secret: ${SECRET:0:10}..."
                    # Use sed with proper escaping to replace the secret
                    ESCAPED_SECRET=$(printf '%s\n' "$SECRET" | sed 's:[][\/.^$*]:\\&:g')
                    MASKED_TEXT=$(echo "$MASKED_TEXT" | sed "s/${ESCAPED_SECRET}/(redacted)/g")
                    SECRET_COUNT=$((SECRET_COUNT + 1))
                  fi
                fi
              done < <(jq -c '.[] | select(.line_data_list != null) | .line_data_list[]' /tmp/credsweeper-results.json 2>/dev/null || echo "")
              
              if [ "$SECRET_COUNT" -gt 0 ]; then
                # Update the agent output with masked text
                echo "Masking $SECRET_COUNT secret(s) in agent output..."
                jq --arg masked_text "$MASKED_TEXT" '.text = $masked_text' "$AGENT_OUTPUT_FILE" > "$AGENT_OUTPUT_FILE.tmp"
                mv "$AGENT_OUTPUT_FILE.tmp" "$AGENT_OUTPUT_FILE"
                
                echo "✓ Secrets masked successfully"
                echo "Found and masked $SECRET_COUNT potential secret(s)"
                
                # Add summary to step summary
                echo "## CredSweeper Secret Masking Summary" >> $GITHUB_STEP_SUMMARY
                echo "" >> $GITHUB_STEP_SUMMARY
                echo "- **Secrets Found**: $SECRET_COUNT" >> $GITHUB_STEP_SUMMARY
                echo "- **Action**: All detected secrets replaced with \`(redacted)\`" >> $GITHUB_STEP_SUMMARY
              else
                echo "No secrets detected by CredSweeper"
              fi
            else
              echo "No secrets detected by CredSweeper"
            fi
            
            # Clean up
            rm -f "$TEMP_FILE" /tmp/credsweeper-results.json
---

<!--

CredSweeper Secret Masking

This shared configuration provides a safe-outputs job that scans agent output
text using Samsung CredSweeper and replaces every detected secret with (redacted).

CredSweeper is a tool to detect and prevent hardcoded credentials like passwords,
API keys, and tokens from being exposed in text, code, and configurations.

GitHub: https://github.com/Samsung/CredSweeper
Documentation: https://samsung.github.io/CredSweeper/

## How It Works

1. The job downloads the agent output artifact automatically
2. Extracts the text content from the JSON output
3. Runs CredSweeper to scan for potential secrets
4. Replaces all detected secrets with "(redacted)"
5. Updates the agent output file with masked content
6. Other safe-output jobs (like create-issue) will use the masked content

The agent can invoke this job by calling the `mask_secrets` tool with text content.
The job processes items with type "mask_secrets" from the agent output.

## Usage

To use this shared workflow, add it to your imports:

```yaml
imports:
  - shared/credsweeper.md
```

This adds a `mask_secrets` safe-output job that the agent can invoke to scan and
mask secrets in its output. The agent should call the mask_secrets tool before
creating issues or PRs to ensure secrets are not exposed.

**Important**: The agent must explicitly call the `mask_secrets` tool for the job
to run. The job does not automatically process all agent output.

The job will:
- Install CredSweeper from PyPI
- Scan the text content for secrets
- Mask any detected credentials with "(redacted)"
- Update the agent output with masked text
- Other safe-output jobs will then use the masked output

## Dependencies

- Python 3 (available on ubuntu-latest runners)
- pip (available on ubuntu-latest runners)
- jq (available on ubuntu-latest runners)

## Permissions

This job requires:
- `contents: read` - To read the repository content

## Environment Variables

The job uses the standard `GH_AW_AGENT_OUTPUT` environment variable which is
automatically set by the safe-outputs framework to point to the agent output file.

## Example Workflow

```yaml
---
on:
  issues:
    types: [opened]
permissions:
  contents: read
  issues: write
engine: claude
imports:
  - shared/credsweeper.md
safe-outputs:
  create-issue:
    title-prefix: "[analysis] "
---

# Analyze Issue

Analyze the issue and create a summary.

**Important**: Before creating the issue, use the `mask_secrets` tool to scan
your response for any potential secrets and mask them.
```

In this example, the agent should call the `mask_secrets` tool with its analysis
text before creating the issue. This ensures that any detected secrets are masked
before the issue is created.

## Security Benefits

- **Prevents credential leaks**: Automatically detects and masks secrets
- **Multiple credential types**: Detects passwords, API keys, tokens, etc.
- **Zero configuration**: Works out of the box with default settings
- **Safe by default**: Runs before other safe-output jobs

## Troubleshooting

### Installation Issues
- Ensure Python 3 and pip are available (they are on ubuntu-latest)
- Check network connectivity for PyPI access

### Scanning Issues  
- Verify agent output file exists and contains JSON
- Check that the text field is present in the output
- Review CredSweeper logs if secrets are not detected

### Performance
- CredSweeper is optimized for text scanning
- Typical scan time is < 5 seconds for most outputs
- Large outputs (>100KB) may take longer

-->
