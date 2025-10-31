---
title: Threat Detection
description: Configure automated threat detection to analyze agent output and code changes for security issues before they are applied.
sidebar:
  order: 650
---

GitHub Agentic Workflows includes automatic threat detection to analyze agent output and code changes for potential security issues before they are applied. When safe outputs are configured, a threat detection job automatically runs to identify prompt injection attempts, secret leaks, and malicious code patches.

## How It Works

Threat detection provides an additional security layer that:

1. **Analyzes Agent Output**: Reviews all safe output items (issues, comments, PRs) for malicious content
2. **Scans Code Changes**: Examines git patches for suspicious patterns, backdoors, and vulnerabilities  
3. **Uses Workflow Context**: Leverages the workflow source to distinguish legitimate actions from threats
4. **Runs Automatically**: Executes after the main agentic job completes but before safe outputs are applied

**Security Architecture:**

```
┌─────────────────┐
│ Agentic Job     │ (Read-only permissions)
│ Generates       │
│ Output & Patches│
└────────┬────────┘
         │ artifacts
         ▼
┌─────────────────┐
│ Threat Detection│ (Analyzes for security issues)
│ Job             │
└────────┬────────┘
         │ approved/blocked
         ▼
┌─────────────────┐
│ Safe Output Jobs│ (Write permissions, only if safe)
│ Create Issues,  │
│ PRs, Comments   │
└─────────────────┘
```

## Default Configuration

Threat detection is **automatically enabled** when safe outputs are configured:

```yaml
safe-outputs:
  create-issue:     # Threat detection enabled automatically
  create-pull-request:
```

The default configuration uses AI-powered analysis with the workflow's Claude or Copilot engine to detect:

- **Prompt Injection**: Malicious instructions attempting to manipulate AI behavior
- **Secret Leaks**: Exposed API keys, tokens, passwords, or credentials
- **Malicious Patches**: Code changes introducing vulnerabilities, backdoors, or suspicious patterns

## Configuration Options

### Basic Enabled/Disabled

Control threat detection with a boolean flag:

```yaml
safe-outputs:
  create-issue:
  threat-detection: true   # Explicitly enable (default when safe-outputs exist)

# Or disable entirely:
safe-outputs:
  create-pull-request:
  threat-detection: false  # Disable threat detection
```

### Advanced Configuration

Use object syntax for fine-grained control:

```yaml
safe-outputs:
  create-issue:
  threat-detection:
    enabled: true                    # Enable/disable detection
    prompt: "Focus on SQL injection" # Additional analysis instructions
    engine:                          # Custom engine configuration
      id: claude
      model: claude-sonnet-4
    steps:                           # Custom detection steps
      - name: Custom Security Check
        run: echo "Running additional checks"
```

**Configuration Fields:**

- **`enabled`** (boolean): Enable or disable threat detection. Default: `true` when safe-outputs exist
- **`prompt`** (string): Additional custom instructions appended to the default threat detection prompt
- **`engine`** (string | object | false): AI engine configuration for detection
  - String format: `"claude"`, `"copilot"`, `"codex"`
  - Object format: Full engine configuration (same as main workflow engine)
  - `false`: Disable AI-based detection, run only custom steps
- **`steps`** (array): Additional GitHub Actions steps to run after AI analysis

## AI-Based Detection (Default)

By default, threat detection uses the same AI engine as your main workflow to analyze output for security threats.

**Example with Default AI Detection:**

```yaml
---
on: push
engine: claude
safe-outputs:
  create-pull-request:
  threat-detection: true  # Uses Claude for analysis
---

# Code Review Agent

Analyze code and create pull requests with improvements.
```

The AI engine receives the workflow source context and analyzes:
- Agent output items (issues, comments, PRs)
- Git patch files with code changes
- Workflow intent and legitimate use cases

**Output Format:**

AI detection produces a structured JSON response:

```json
{
  "prompt_injection": false,
  "secret_leak": false,
  "malicious_patch": false,
  "reasons": []
}
```

If any threat is detected (`true`), the workflow fails and safe outputs are blocked.

## Custom Prompt Instructions

Enhance AI analysis with domain-specific guidance:

```yaml
safe-outputs:
  create-pull-request:
  threat-detection:
    prompt: |
      Focus on these additional security concerns:
      - SQL injection vulnerabilities in database queries
      - Cross-site scripting (XSS) in user input handling
      - Unsafe deserialization patterns
      - Hardcoded credentials in configuration files
```

The custom prompt is appended to the default threat detection instructions, providing specialized context for your workflow's domain.

## Custom Engine Configuration

Override the main workflow engine for threat detection:

**String Format:**

```yaml
safe-outputs:
  create-pull-request:
  threat-detection:
    engine: copilot  # Use Copilot instead of main workflow engine
```

**Object Format:**

```yaml
safe-outputs:
  create-pull-request:
  threat-detection:
    engine:
      id: claude
      model: claude-sonnet-4
      max-turns: 3
```

**Disable AI Engine:**

```yaml
safe-outputs:
  create-pull-request:
  threat-detection:
    engine: false    # No AI analysis, only custom steps
    steps:
      - name: Custom Scanning
        run: |
          # Run your own security tools
          ./security-scan.sh
```

## Custom Detection Steps

Add specialized security scanning tools alongside or instead of AI detection:

```yaml
safe-outputs:
  create-pull-request:
  threat-detection:
    steps:
      - name: Run Security Scanner
        run: |
          echo "Scanning agent output for threats..."
          # Your custom security logic here

      - name: Check for Secrets
        uses: trufflesecurity/trufflehog@main
        with:
          path: /tmp/gh-aw/threat-detection/
```

**Available Artifacts:**

Custom steps have access to these downloaded artifacts:

- `/tmp/gh-aw/threat-detection/prompt.txt` - Workflow prompt
- `/tmp/gh-aw/threat-detection/agent_output.json` - Safe output items
- `/tmp/gh-aw/threat-detection/aw.patch` - Git patch file

**Execution Order:**

1. Download artifacts (prompt, output, patch)
2. Run AI-based analysis (if engine not disabled)
3. Execute custom steps
4. Upload detection log artifact

## Example: LlamaGuard Integration

Use Ollama with LlamaGuard 3 for specialized threat detection:

```yaml
---
on: push
engine: copilot
safe-outputs:
  create-pull-request:
  threat-detection:
    steps:
      - name: Ollama LlamaGuard 3 Scan
        uses: actions/github-script@v7
        with:
          script: |
            const fs = require('fs');
            
            // Install Ollama
            await exec.exec('curl', ['-fsSL', 'https://ollama.com/install.sh', '-o', '/tmp/install.sh']);
            await exec.exec('sh', ['/tmp/install.sh']);
            
            // Start Ollama service
            exec.exec('ollama', ['serve'], { detached: true });
            
            // Wait for service
            let ready = false;
            for (let i = 0; i < 30; i++) {
              try {
                await exec.exec('curl', ['-f', 'http://localhost:11434/api/version'], { silent: true });
                ready = true;
                break;
              } catch (e) {
                await new Promise(r => setTimeout(r, 1000));
              }
            }
            
            if (!ready) {
              core.setFailed('Ollama service failed to start');
              return;
            }
            
            // Pull LlamaGuard model
            await exec.exec('ollama', ['pull', 'llama-guard3:1b']);
            
            // Scan agent output
            const outputPath = '/tmp/gh-aw/threat-detection/agent_output.json';
            if (fs.existsSync(outputPath)) {
              const content = fs.readFileSync(outputPath, 'utf8');
              
              const response = await exec.getExecOutput('curl', [
                '-X', 'POST',
                'http://localhost:11434/api/chat',
                '-H', 'Content-Type: application/json',
                '-d', JSON.stringify({
                  model: 'llama-guard3:1b',
                  messages: [{ role: 'user', content }],
                  stream: false
                })
              ]);
              
              const result = JSON.parse(response.stdout);
              const output = result.message?.content || '';
              
              // Check if safe
              const isSafe = output.toLowerCase().trim() === 'safe' || output.includes('s8');
              
              if (!isSafe) {
                core.setFailed(`LlamaGuard detected threat: ${output}`);
              } else {
                core.info('✅ Content appears safe');
              }
            }

timeout_minutes: 20  # Allow time for model download
---

# Code Review Agent

Analyze and improve code with LlamaGuard threat scanning.
```

:::tip
For a complete LlamaGuard implementation, see `.github/workflows/shared/ollama-threat-scan.md` in the repository.
:::

## Combined AI and Custom Detection

Use both AI analysis and custom tools for defense-in-depth:

```yaml
safe-outputs:
  create-pull-request:
  threat-detection:
    prompt: "Check for authentication bypass vulnerabilities"
    engine:
      id: claude
      model: claude-sonnet-4
    steps:
      - name: Static Analysis
        run: |
          # Run static analysis tool
          semgrep --config auto /tmp/gh-aw/threat-detection/

      - name: Secret Scanner
        uses: trufflesecurity/trufflehog@main
        with:
          path: /tmp/gh-aw/threat-detection/aw.patch
```

This configuration:
1. Uses Claude with custom prompt for AI analysis
2. Runs Semgrep for static code analysis
3. Scans for exposed secrets with TruffleHog

## Error Handling

**When Threats Are Detected:**

The threat detection job fails with a clear error message and safe output jobs are skipped:

```
❌ Threat detected: Potential SQL injection in code changes
Reasons:
- Unsanitized user input in database query
- Missing parameterized query pattern
```

**When Detection Fails:**

If the detection process itself fails (e.g., network issues, tool errors), the workflow stops and safe outputs are not applied. This fail-safe approach prevents potentially malicious content from being processed.

## Best Practices

### When to Use AI Detection

**Use AI-based detection when:**
- Analyzing natural language content (issues, comments, discussions)
- Detecting sophisticated prompt injection attempts
- Understanding context-specific security risks
- Identifying intent-based threats

### When to Use Custom Steps

**Add custom steps when:**
- Integrating specialized security tools (Semgrep, Snyk, TruffleHog)
- Enforcing organization-specific security policies
- Scanning for domain-specific vulnerabilities
- Meeting compliance requirements

### Performance Considerations

- **AI Analysis**: Typically completes in 10-30 seconds
- **Custom Tools**: Varies by tool (LlamaGuard: 5-15 minutes with model download)
- **Timeout**: Set appropriate `timeout_minutes` for custom tools
- **Artifact Size**: Large patches may require truncation for analysis

### Security Recommendations

1. **Defense in Depth**: Use both AI and custom detection for critical workflows
2. **Regular Updates**: Keep custom security tools and models up to date
3. **Test Thoroughly**: Validate detection with known malicious samples
4. **Monitor False Positives**: Review blocked outputs to refine detection logic
5. **Document Rationale**: Comment why specific detection rules exist

## Troubleshooting

### AI Detection Always Fails

**Symptom**: Every workflow execution reports threats

**Solutions**:
- Review custom prompt for overly strict instructions
- Check if legitimate workflow patterns trigger detection
- Adjust prompt to provide better context
- Use `threat-detection.enabled: false` temporarily to test

### Custom Steps Not Running

**Symptom**: Steps in `threat-detection.steps` don't execute

**Check**:
- Verify YAML indentation is correct
- Ensure steps array is properly formatted
- Review workflow compilation output for errors
- Check if AI detection failed before custom steps

### Large Patches Cause Timeouts

**Symptom**: Detection times out with large code changes

**Solutions**:
- Increase `timeout_minutes` in workflow frontmatter
- Configure `max-patch-size` to limit patch size
- Truncate content before analysis in custom steps
- Split large changes into smaller PRs

### False Positives

**Symptom**: Legitimate content flagged as malicious

**Solutions**:
- Refine custom prompt with specific exclusions
- Adjust custom detection tool thresholds
- Add workflow context explaining legitimate patterns
- Review detection logs to understand trigger patterns

## Related Documentation

- [Safe Outputs Reference](/gh-aw/reference/safe-outputs/) - Complete safe outputs configuration
- [Security Guide](/gh-aw/guides/security/) - Overall security best practices
- [Custom Safe Outputs](/gh-aw/guides/custom-safe-outputs/) - Creating custom output types
- [Frontmatter Reference](/gh-aw/reference/frontmatter/) - All configuration options
