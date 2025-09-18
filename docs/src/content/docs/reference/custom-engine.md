---
title: Custom Engine
description: Complete guide to the custom agentic engine that allows you to define your own GitHub Actions steps instead of using AI interpretation.
sidebar:
  order: 6
---

The custom engine is an advanced feature that allows you to define completely custom GitHub Actions steps instead of using AI interpretation. This is perfect for deterministic workflows, hybrid approaches, or when you need specific behaviors that AI engines can't provide.

## Overview

Unlike the Claude and Codex engines that interpret natural language using AI, the custom engine executes user-defined GitHub Actions steps directly. The markdown content of your workflow becomes available as a prompt file that your custom steps can read and process as needed.

```yaml
engine: custom
```

**Key Characteristics:**
- **No AI interpretation** - your steps run exactly as defined
- **Direct GitHub Actions execution** - standard step syntax
- **Access to workflow content** - markdown available as environment variable
- **Hybrid workflows** - combine deterministic and AI-powered steps
- **Full control** - you define exactly what happens

## Basic Configuration

### Simple Setup

```yaml
---
engine: custom
---

# My Custom Workflow

This markdown content will be available to your custom steps as a prompt file.

Your steps can read this content and process it however you need.
```

### Extended Configuration

```yaml
---
engine:
  id: custom
  max-turns: 10                    # Optional: for consistency with other engines
  env:                            # Optional: custom environment variables
    DEBUG_MODE: "true"
    CUSTOM_API_ENDPOINT: "https://api.example.com"
  steps:                          # Required: your custom GitHub Actions steps
    - name: Setup Node.js
      uses: actions/setup-node@v4
      with:
        node-version: "18"
    - name: Run custom logic
      run: |
        echo "Processing workflow content..."
        cat $GITHUB_AW_PROMPT
        # Your custom logic here
---

# My Advanced Custom Workflow

This content is available in the $GITHUB_AW_PROMPT file.
```

## Available Environment Variables

Your custom steps automatically have access to several environment variables:

### Core Variables

- **`$GITHUB_AW_PROMPT`**: Path to the generated prompt file (`/tmp/aw-prompts/prompt.txt`)
  - Contains the markdown content from your workflow
  - This is the natural language instructions that would normally be sent to an AI processor
  - Your custom steps can read this file to access the workflow's markdown content programmatically

### Safe Outputs Variables

When [safe-outputs](/gh-aw/reference/safe-outputs/) are configured, additional variables are available:

- **`$GITHUB_AW_SAFE_OUTPUTS`**: Path to the safe outputs file
  - Used for writing structured output that gets processed automatically
  - Required when using safe-outputs features

- **`$GITHUB_AW_SAFE_OUTPUTS_STAGED`**: Set to `"true"` when staged mode is enabled
  - Available when `safe-outputs.staged: true` is configured

### Engine Configuration Variables

- **`$GITHUB_AW_MAX_TURNS`**: Maximum number of turns/iterations
  - Available when `max-turns` is configured in engine config
  - Useful for consistency with other engines

### Custom Environment Variables

Any variables defined in the `env` section of your engine configuration:

```yaml
engine:
  id: custom
  env:
    DEBUG_MODE: "true"
    API_KEY: ${{ secrets.MY_API_KEY }}
    REGION: "us-west-2"
  steps:
    - name: Use custom variables
      run: |
        echo "Debug mode: $DEBUG_MODE"
        echo "API Key length: ${#API_KEY}"
        echo "Region: $REGION"
```

## Working with Safe Outputs

The custom engine fully supports [safe outputs](/gh-aw/reference/safe-outputs/) for secure GitHub operations without requiring write permissions for your main workflow.

### Using the Safe Outputs MCP Server

When safe-outputs are configured, an MCP server is automatically available:

```yaml
---
engine: custom
safe-outputs:
  create-issue:
tools:
  safe-outputs: {}  # Enables the safe outputs MCP server
---

# Issue Creation Workflow

Analyze the repository and create issues for any problems found.
```

Your custom steps can use the MCP server to interact with safe outputs:

```yaml
steps:
  - name: Install MCP client
    run: npm install -g @anthropic-ai/claude-code
  
  - name: Use safe outputs MCP
    run: |
      # The MCP server is available at /tmp/safe-outputs/mcp-server.cjs
      # You can interact with it using MCP-compatible tools
      echo "Safe outputs available at: $GITHUB_AW_SAFE_OUTPUTS"
```

### Direct File Writing

Alternatively, you can write directly to the safe outputs file:

```yaml
---
engine: custom
safe-outputs:
  create-issue:
---

# Direct File Writing Example

Process data and create issues based on findings.
```

```yaml
steps:
  - name: Analyze and create issues
    run: |
      # Read the workflow content
      echo "Processing instructions:"
      cat $GITHUB_AW_PROMPT
      
      # Write issue data directly to the safe outputs file
      cat > $GITHUB_AW_SAFE_OUTPUTS << 'EOF'
      {
        "issues": [
          {
            "title": "Found potential security issue",
            "body": "During analysis, I discovered...",
            "labels": ["security", "bug"]
          }
        ]
      }
      EOF
      
      echo "Issue data written to $GITHUB_AW_SAFE_OUTPUTS"
```

## Advanced Examples

### Using actions/ai-inference

You can integrate the `actions/ai-inference` action for hybrid workflows that combine deterministic steps with AI capabilities:

```yaml
---
engine:
  id: custom
  env:
    GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
  steps:
    - name: Setup analysis environment
      run: |
        echo "Preparing analysis environment..."
        mkdir -p /tmp/analysis
        
    - name: AI-powered code analysis
      uses: actions/ai-inference@v1
      with:
        model: "gpt-4"
        prompt: |
          Analyze the following repository content and provide insights:
          
          $(cat $GITHUB_AW_PROMPT)
          
          Focus on:
          1. Code quality issues
          2. Security vulnerabilities
          3. Performance opportunities
        output-file: /tmp/analysis/ai-results.json
      env:
        OPENAI_API_KEY: ${{ secrets.OPENAI_API_KEY }}
    
    - name: Process AI results
      run: |
        echo "AI analysis complete. Processing results..."
        
        # Read AI results
        if [ -f /tmp/analysis/ai-results.json ]; then
          echo "AI Results:"
          cat /tmp/analysis/ai-results.json
          
          # Transform results for safe outputs if configured
          if [ -n "$GITHUB_AW_SAFE_OUTPUTS" ]; then
            echo "Converting to safe outputs format..."
            # Process and write to $GITHUB_AW_SAFE_OUTPUTS
          fi
        else
          echo "No AI results found"
        fi
---

# Hybrid AI Analysis Workflow

This workflow combines deterministic setup and processing with AI-powered analysis.

Please analyze the current repository for:
- Code quality issues
- Security vulnerabilities  
- Performance optimization opportunities
- Documentation gaps

Provide specific, actionable recommendations for each finding.
```

### Multi-Step Processing Pipeline

```yaml
---
engine:
  id: custom
  env:
    WORKSPACE: /tmp/custom-workspace
  steps:
    - name: Initialize workspace
      run: |
        mkdir -p $WORKSPACE
        echo "Workspace initialized at $WORKSPACE"
        
    - name: Extract requirements from prompt
      run: |
        echo "Extracting requirements from workflow content..."
        cat $GITHUB_AW_PROMPT > $WORKSPACE/original-prompt.md
        
        # Parse specific requirements
        grep -E "^-|^\*|^[0-9]+\." $GITHUB_AW_PROMPT > $WORKSPACE/requirements.txt || true
        echo "Requirements extracted to $WORKSPACE/requirements.txt"
        
    - name: Setup custom tools
      uses: actions/setup-python@v4
      with:
        python-version: '3.11'
        
    - name: Install dependencies
      run: |
        pip install requests pyyaml jinja2
        
    - name: Process requirements
      run: |
        python << 'EOF'
        import json
        import os
        import yaml
        
        # Read the prompt content
        with open(os.environ['GITHUB_AW_PROMPT'], 'r') as f:
            prompt_content = f.read()
            
        # Read extracted requirements
        workspace = os.environ['WORKSPACE']
        try:
            with open(f'{workspace}/requirements.txt', 'r') as f:
                requirements = f.read().strip().split('\n')
        except:
            requirements = []
            
        # Process and generate outputs
        results = {
            'prompt_length': len(prompt_content),
            'requirements_count': len([r for r in requirements if r.strip()]),
            'processed_at': '$(date -Iseconds)',
            'workspace': workspace
        }
        
        # Save results
        with open(f'{workspace}/results.json', 'w') as f:
            json.dump(results, f, indent=2)
            
        print("Processing complete!")
        print(f"Results: {results}")
        EOF
        
    - name: Generate final output
      run: |
        echo "Final processing step..."
        echo "Results summary:"
        cat $WORKSPACE/results.json
        
        # If safe outputs are configured, write to the output file
        if [ -n "$GITHUB_AW_SAFE_OUTPUTS" ]; then
          echo "Writing to safe outputs..."
          cat $WORKSPACE/results.json > $GITHUB_AW_SAFE_OUTPUTS
        fi
---

# Multi-Step Processing Workflow

This workflow demonstrates a complex processing pipeline:

1. **Parse requirements** from this markdown content
2. **Set up processing environment** with Python and required libraries  
3. **Analyze content** using custom Python scripts
4. **Generate structured output** for downstream processing

## Requirements to Process

- Analyze repository structure
- Check code quality metrics
- Generate summary report
- Create actionable recommendations
```

### Integration with External APIs

```yaml
---
engine:
  id: custom
  env:
    API_BASE_URL: "https://api.example.com"
    API_KEY: ${{ secrets.EXTERNAL_API_KEY }}
  steps:
    - name: Read workflow instructions
      id: parse-instructions
      run: |
        echo "Parsing workflow instructions..."
        
        # Extract API endpoints to call from the prompt
        endpoints=$(grep -o 'https://[^[:space:]]*' $GITHUB_AW_PROMPT || echo "")
        
        echo "endpoints=$endpoints" >> $GITHUB_OUTPUT
        echo "Found endpoints: $endpoints"
        
    - name: Call external APIs
      run: |
        echo "Calling external APIs..."
        
        # Read the workflow content for context
        echo "Workflow context:"
        cat $GITHUB_AW_PROMPT
        echo "---"
        
        # Make API calls based on the prompt content
        curl -H "Authorization: Bearer $API_KEY" \
             -H "Content-Type: application/json" \
             -d '{"prompt": "'"$(cat $GITHUB_AW_PROMPT | base64 -w 0)"'"}' \
             "$API_BASE_URL/analyze" \
             > /tmp/api-response.json
             
        echo "API response received"
        
    - name: Process API response
      run: |
        if [ -f /tmp/api-response.json ]; then
          echo "Processing API response..."
          cat /tmp/api-response.json | jq '.'
          
          # Convert response for safe outputs if needed
          if [ -n "$GITHUB_AW_SAFE_OUTPUTS" ]; then
            echo "Converting API response to safe outputs format..."
            jq '{
              "analysis": .result,
              "confidence": .confidence,
              "recommendations": .recommendations
            }' /tmp/api-response.json > $GITHUB_AW_SAFE_OUTPUTS
          fi
        else
          echo "No API response found"
        fi
---

# External API Integration Workflow

This workflow integrates with external APIs for enhanced processing:

Call the analysis API with this content and return:
- Summary of findings
- Confidence scores
- Actionable recommendations

The API should analyze for:
- Code structure and patterns
- Best practices compliance
- Potential improvements
```

## Environment Variable Merging

If your custom steps already define environment variables, the custom engine intelligently merges them:

```yaml
engine:
  id: custom
  env:
    GLOBAL_VAR: "global-value"
  steps:
    - name: Step with existing env
      run: echo "Both variables available: $GLOBAL_VAR and $STEP_VAR"
      env:
        STEP_VAR: "step-value"
        # GLOBAL_VAR is automatically added
```

**Merging Rules:**
- Custom engine variables are added to existing step `env` sections
- If a step doesn't have an `env` section, one is created
- Step-level variables take precedence over engine-level variables
- Invalid `env` sections are replaced with engine variables

## Best Practices

### 1. Structure Your Steps Logically

```yaml
steps:
  # Setup phase
  - name: Initialize environment
    run: mkdir -p /tmp/workspace
    
  # Processing phase  
  - name: Process workflow content
    run: |
      cat $GITHUB_AW_PROMPT > /tmp/workspace/prompt.md
      # Processing logic here
      
  # Output phase
  - name: Generate results
    run: |
      # Generate final outputs
      echo "Results ready"
```

### 2. Handle Errors Gracefully

```yaml
steps:
  - name: Safe processing
    run: |
      set -e  # Exit on error
      
      if [ ! -f "$GITHUB_AW_PROMPT" ]; then
        echo "Error: Prompt file not found"
        exit 1
      fi
      
      # Your processing logic
      echo "Processing complete"
```

### 3. Use Conditional Logic

```yaml
steps:
  - name: Conditional processing
    run: |
      # Check if safe outputs are configured
      if [ -n "$GITHUB_AW_SAFE_OUTPUTS" ]; then
        echo "Safe outputs enabled, writing results..."
        echo '{"status": "complete"}' > $GITHUB_AW_SAFE_OUTPUTS
      else
        echo "Safe outputs not configured, using standard output"
      fi
```

### 4. Leverage Existing Actions

```yaml
steps:
  - name: Setup tools
    uses: actions/setup-node@v4
    with:
      node-version: '20'
      
  - name: Install dependencies  
    run: npm install -g some-cli-tool
    
  - name: Use tools with prompt content
    run: |
      some-cli-tool --input $GITHUB_AW_PROMPT --output /tmp/results.json
```

## Debugging and Troubleshooting

### View Available Environment Variables

```yaml
steps:
  - name: Debug environment
    run: |
      echo "=== Custom Engine Environment Variables ==="
      echo "GITHUB_AW_PROMPT: $GITHUB_AW_PROMPT"
      echo "GITHUB_AW_SAFE_OUTPUTS: $GITHUB_AW_SAFE_OUTPUTS"
      echo "GITHUB_AW_MAX_TURNS: $GITHUB_AW_MAX_TURNS"
      echo "GITHUB_AW_SAFE_OUTPUTS_STAGED: $GITHUB_AW_SAFE_OUTPUTS_STAGED"
      echo "=== Prompt Content ==="
      if [ -f "$GITHUB_AW_PROMPT" ]; then
        cat $GITHUB_AW_PROMPT
      else
        echo "Prompt file not found"
      fi
```

### Validate File Permissions

```yaml
steps:
  - name: Check file access
    run: |
      echo "Checking file permissions..."
      ls -la $GITHUB_AW_PROMPT
      
      if [ -n "$GITHUB_AW_SAFE_OUTPUTS" ]; then
        echo "Safe outputs file: $GITHUB_AW_SAFE_OUTPUTS"
        touch $GITHUB_AW_SAFE_OUTPUTS
        ls -la $GITHUB_AW_SAFE_OUTPUTS
      fi
```

## Migration from AI Engines

### From Claude Engine

```yaml
# Before (Claude)
engine: claude

# After (Custom with AI integration)
engine:
  id: custom
  steps:
    - name: AI processing
      uses: actions/ai-inference@v1
      with:
        model: "claude-3-sonnet"
        prompt: "$(cat $GITHUB_AW_PROMPT)"
```

### From Codex Engine

```yaml
# Before (Codex)
engine:
  id: codex
  config: |
    [custom_section]
    key = "value"

# After (Custom)
engine:
  id: custom
  env:
    CUSTOM_CONFIG_KEY: "value"
  steps:
    - name: Setup configuration
      run: |
        echo "key=value" > /tmp/config.env
        source /tmp/config.env
```

## Related Documentation

- [AI Engines](/gh-aw/reference/engines/) - Overview of all available engines
- [Safe Output Processing](/gh-aw/reference/safe-outputs/) - Secure output handling
- [Frontmatter Options](/gh-aw/reference/frontmatter/) - Complete configuration reference
- [Tools Configuration](/gh-aw/reference/tools/) - Available tools and MCP servers