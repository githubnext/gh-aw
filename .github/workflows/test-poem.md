---
on:
  workflow_dispatch:
  issues:
    types: [opened, edited]
  pull_request:
    types: [opened, edited, synchronize]

safe-outputs:
  missing-tool:
    max: 5
  staged: true

engine:
  id: custom
  steps:
    - name: Gather Repository Context
      run: |
        echo "Gathering context about current changes and issues for poem generation..."
        
        # Check if we're in a PR context
        if [ "${{ github.event_name }}" = "pull_request" ]; then
          echo "Processing pull request context..."
          PR_TITLE="${{ github.event.pull_request.title }}"
          PR_BODY="${{ github.event.pull_request.body }}"
          PR_NUMBER="${{ github.event.pull_request.number }}"
          echo "PR #$PR_NUMBER: $PR_TITLE" > /tmp/context.txt
          echo "$PR_BODY" >> /tmp/context.txt
        elif [ "${{ github.event_name }}" = "issues" ]; then
          echo "Processing issue context..."
          ISSUE_TITLE="${{ github.event.issue.title }}"
          ISSUE_BODY="${{ github.event.issue.body }}"
          ISSUE_NUMBER="${{ github.event.issue.number }}"
          echo "Issue #$ISSUE_NUMBER: $ISSUE_TITLE" > /tmp/context.txt
          echo "$ISSUE_BODY" >> /tmp/context.txt
        else
          echo "Processing general repository context..."
          echo "Manual workflow dispatch for repository: ${{ github.repository }}" > /tmp/context.txt
          echo "Triggered by: ${{ github.actor }}" >> /tmp/context.txt
          echo "Branch: ${{ github.ref_name }}" >> /tmp/context.txt
        fi
        
        echo "Context gathered successfully"
        
    - name: Generate Poem About Changes
      run: |
        # Read the context
        CONTEXT=""
        if [ -f /tmp/context.txt ]; then
          CONTEXT=$(cat /tmp/context.txt)
        fi
        
        # Create a poem based on the context
        cat > /tmp/poem.txt << 'EOF'
        In the realm of code where changes flow,
        Through GitHub's halls where pull requests go,
        A workflow runs with purpose clear,
        To test our safe outputs here.
        
        When issues rise and PRs appear,
        This test-poem workflow draws near,
        With missing tools it tells its tale,
        Of features lost beyond the pale.
        
        The context speaks of work in progress,
        Of bugs to fix and code to bless,
        Each commit tells a story true,
        Of what the developers pursue.
        
        In staged mode safe, no harm is done,
        Just testing how the workflows run,
        A poem born from code's embrace,
        To validate this testing space.
        
        So here we stand with workflow strong,
        Our test-poem sings its testing song,
        May safe outputs guide the way,
        Through every test and every day.
        EOF
        
        echo "Poem generated successfully!"
        cat /tmp/poem.txt
        
    - name: Report Missing Tool for Poetry Generation
      run: |
        # Simulate missing a specialized poetry generation tool
        echo '{"type": "missing-tool", "tool": "advanced-poetry-generator", "reason": "Advanced AI poetry generation tool needed for context-aware poem creation about code changes and repository state", "alternatives": "Currently using basic template-based poem generation. Could integrate with Claude, GPT, or specialized poetry APIs for more sophisticated verse creation", "context": "test-poem workflow poetry generation"}' >> $GITHUB_AW_SAFE_OUTPUTS
        
    - name: Report Missing Tool for Sentiment Analysis
      run: |
        # Report another missing tool for sentiment analysis of changes
        echo '{"type": "missing-tool", "tool": "sentiment-analyzer", "reason": "Sentiment analysis tool needed to analyze the emotional tone of issue descriptions and pull request changes to generate mood-appropriate poetry", "alternatives": "Could use natural language processing libraries like spaCy, NLTK, or cloud services like Azure Text Analytics, AWS Comprehend", "context": "test-poem workflow sentiment analysis for poetic tone"}' >> $GITHUB_AW_SAFE_OUTPUTS
        
    - name: Display Generated Poem
      run: |
        echo "## Generated Poem 📝" >> $GITHUB_STEP_SUMMARY
        echo "" >> $GITHUB_STEP_SUMMARY
        echo '```text' >> $GITHUB_STEP_SUMMARY
        if [ -f /tmp/poem.txt ]; then
          cat /tmp/poem.txt >> $GITHUB_STEP_SUMMARY
        else
          echo "No poem was generated" >> $GITHUB_STEP_SUMMARY
        fi
        echo '```' >> $GITHUB_STEP_SUMMARY
        echo "" >> $GITHUB_STEP_SUMMARY
        echo "**Context:** ${{ github.event_name }} triggered by ${{ github.actor }}" >> $GITHUB_STEP_SUMMARY
        
    - name: Verify Safe Output File
      run: |
        echo "Generated safe output entries:"
        if [ -f "$GITHUB_AW_SAFE_OUTPUTS" ]; then
          cat "$GITHUB_AW_SAFE_OUTPUTS"
        else
          echo "No safe outputs file found"
        fi

permissions: read-all
---

# Test Poem Workflow

This workflow generates a poem about the current changes or issues in the repository and demonstrates the `missing-tool` safe output functionality.

## Purpose

This workflow validates the missing-tool safe output type by:
- Triggering on workflow dispatch, issue events, and pull request events
- Generating a contextual poem about the current repository state
- Reporting missing tools needed for advanced poetry generation
- Using staged mode to prevent actual GitHub interactions
- Demonstrating how workflows can creatively respond to repository events

## Trigger Events

- **workflow_dispatch**: Manual execution for testing
- **issues.opened/edited**: Responds to issue creation and updates
- **pull_request.opened/edited/synchronize**: Responds to PR events

## Safe Output Configuration

- **staged: true**: Prevents real GitHub interactions
- **max: 5**: Allows up to 5 missing tool reports per workflow run

## Custom Engine Implementation

The workflow uses a custom engine with GitHub Actions steps to:
1. Gather context about the current repository state (issues, PRs, or general info)
2. Generate a poem about the changes using template-based approach
3. Report missing tools needed for advanced poetry generation and sentiment analysis
4. Display the generated poem in the workflow summary
5. Verify the safe outputs were generated correctly

This demonstrates how custom engines can leverage the safe output system for creative applications while reporting limitations that prevent full task completion.

## Example Use Cases

- **Issue Opened**: Creates a poem about the new issue and its context
- **Pull Request**: Generates verse about the proposed changes
- **Manual Dispatch**: Creates a general poem about the repository state

The workflow showcases how agentic workflows can be both functional and creative while properly reporting their limitations through the safe output system.