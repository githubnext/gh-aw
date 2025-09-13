---
on:
  workflow_dispatch:
  issues:
    types: [opened]

permissions: read-all

safe-outputs:
  add-issue-comment:
    max: 1
    target: "*"
  staged: true

engine:
  id: custom
  steps:
    - name: Setup Node.js
      uses: actions/setup-node@v4
      with:
        node-version: '20'
        
    - name: Install GenAIScript CLI
      run: |
        npm install -g genaiscript
        
    - name: Create GenAIScript Markdown File
      run: |
        cat > analysis_script.genai.md << 'EOF'
        ---
        title: "Issue Analysis Script"
        description: "Analyze GitHub issue and provide helpful insights"
        model: gpt-4
        ---

        # Issue Analysis

        You are an expert GitHub issue analyst. Your task is to analyze the provided issue and generate helpful insights.

        ## Context

        - **Repository**: ${{ github.repository }}
        - **Issue Number**: #${{ github.event.issue.number }}
        - **Actor**: ${{ github.actor }}
        - **Workflow**: ${{ github.workflow }}

        ## Analysis Instructions

        Please analyze this issue and provide:

        1. **Issue Classification**: Categorize the issue (bug, feature request, question, etc.)
        2. **Priority Assessment**: Suggest priority level based on content
        3. **Required Information**: Identify any missing information needed
        4. **Suggested Actions**: Recommend next steps for the maintainers
        5. **Similar Issues**: Note if this seems similar to common patterns

        ## Output Format

        Provide your analysis in the following JSON format:

        ```json
        {
          "classification": "bug|feature|question|documentation|other",
          "priority": "low|medium|high|critical",
          "missing_info": ["list", "of", "missing", "details"],
          "suggested_actions": ["action1", "action2"],
          "comment": "Thank you for opening issue #${{ github.event.issue.number }}! Our automated analysis is processing your request and a maintainer will review it soon."
        }
        ```

        The comment should be friendly, helpful, and provide value to the issue author.
        EOF
        
    - name: Run GenAIScript Analysis
      run: |
        genaiscript run analysis_script.genai.md > analysis_output.json
        
    - name: Process Analysis and Generate Safe Output
      run: |
        # Generate safe output for issue comment
        echo "{\"type\": \"add-issue-comment\", \"body\": \"Thank you for opening issue #${{ github.event.issue.number }}! This issue has been automatically analyzed by our GenAIScript workflow. A maintainer will review your request soon.\"}" >> $GITHUB_AW_SAFE_OUTPUTS
        
    - name: Verify Safe Output File
      run: |
        echo "Generated safe output entries:"
        if [ -f "$GITHUB_AW_SAFE_OUTPUTS" ]; then
          cat "$GITHUB_AW_SAFE_OUTPUTS"
        else
          echo "No safe outputs file found"
        fi

timeout_minutes: 10
---

# GenAIScript Issue Analyzer

This workflow demonstrates using GenAIScript CLI as a custom engine for analyzing GitHub issues using the markdown script format.

## Overview

GenAIScript is a powerful tool for creating AI-powered scripts using markdown files. This workflow shows how to integrate GenAIScript with GitHub Agentic Workflows to:

- Analyze newly opened GitHub issues
- Use GenAIScript's markdown-based scripting format
- Generate helpful automated responses
- Demonstrate custom engine integration patterns

## GenAIScript Features Demonstrated

### Markdown Script Format
The workflow creates a `.genai.md` file that uses GenAIScript's markdown scripting format:
- YAML frontmatter for configuration (title, description, model)
- Markdown content with prompts and instructions
- Template variables for GitHub context
- Structured output requirements

### AI Model Integration
- Uses GPT-4 model through GenAIScript
- Processes GitHub issue data (title, body, author, labels)
- Generates structured JSON output for further processing

### Custom Engine Pattern
This workflow demonstrates the custom engine pattern for integrating external AI tools:
1. **Setup**: Install required dependencies (Node.js, GenAIScript CLI)
2. **Script Creation**: Generate markdown script files dynamically
3. **Execution**: Run GenAIScript with the markdown script
4. **Processing**: Parse output and generate safe outputs

## Trigger Events

- **workflow_dispatch**: Manual execution for testing
- **issues.opened**: Automatic analysis of new issues

## Safe Output Configuration

- **staged: true**: Prevents real GitHub interactions during testing
- **target: "*"**: Allows comments on any issue
- **max: 1**: Limits to one comment per workflow run

## GenAIScript Markdown Script Structure

The generated script follows GenAIScript conventions:

```markdown
---
title: "Issue Analysis Script"
description: "Analyze GitHub issue and provide helpful insights"
model: gpt-4
---

# Issue Analysis

[Prompt and instructions for the AI model]

## Issue Details
[Template variables with GitHub context]

## Analysis Instructions
[Specific tasks and requirements]

## Output Format
[Structured output specification]
```

## Benefits of GenAIScript Integration

1. **Markdown-Native**: Scripts are written in familiar markdown format
2. **Model Flexibility**: Support for different AI models (GPT-4, Claude, etc.)
3. **Template Variables**: Easy integration with GitHub context data
4. **Structured Output**: JSON output for programmatic processing
5. **Version Control**: GenAI scripts can be version controlled alongside code

## Usage Examples

This pattern can be extended for various use cases:
- Code review automation
- Documentation generation
- Bug triage and classification
- Feature request analysis
- Security vulnerability assessment

## Prerequisites

For this workflow to run successfully:
- Node.js 20+ must be available
- GenAIScript CLI must be installable via npm
- AI model API access must be configured (typically via environment variables)
- Appropriate GitHub permissions for the workflow

## Security Considerations

- Uses read-only permissions by default
- All outputs go through safe-outputs validation
- Staged mode prevents unintended GitHub interactions
- External dependencies are explicitly declared and installed