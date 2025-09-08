---
name: AI Inference with GitHub Models
on:
  workflow_dispatch:
    inputs:
      issue_number:
        description: 'The number of the issue to analyze'
        required: true
  issues:
    types: [opened]

permissions:
  contents: read
  models: read

engine:
  id: custom
  max-turns: 3
  steps:
    - name: Setup AI Inference with GitHub Models
      uses: actions/ai-inference@v1
      id: ai_inference
      with:
        # Use gpt-4o-mini model
        model: gpt-4o-mini
        # Use the provided prompt or create one based on the event
        prompt-file: ${{ env.GITHUB_AW_PROMPT }}
        # Configure the AI inference settings
        max-tokens: 1000
      env:
        GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}

    - name: Create Issue Comment
      uses: actions/github-script@v7
      env:
        AI_RESPONSE: ${{ steps.ai_inference.outputs.response }}
      with:
        script: |
          const fs = require('fs');          
          const issueNumber = context.eventName === 'issues' 
            ? context.issue.number 
            : context.payload.inputs.issue_number;
          const aiResponse = process.env.AI_RESPONSE;          
          const safeOutput = {
            type: "add-issue-comment",
            issue_number: issueNumber,
            body: aiResponse
          };          
          fs.appendFileSync(process.env.GITHUB_AW_SAFE_OUTPUTS, JSON.stringify(safeOutput) + '\n');

safe-outputs:
  add-issue-comment:
    max: 1
    target: "*"
---

Summarize the issue inlined below and provide suggestions for next steps.

---

${{ needs.task.outputs.text }}