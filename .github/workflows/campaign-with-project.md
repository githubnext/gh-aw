---
name: Multi-Agent Research Campaign
engine: copilot

on:
  workflow_dispatch:
    inputs:
      research_topics:
        description: 'Comma-separated list of research topics'
        required: true
        default: 'AI safety, Machine learning ethics, Responsible AI'

campaign:
  project:
    name: "Research Campaign - ${{ github.run_id }}"
    view: board
    status-field: "Status"
    agent-field: "Agent"
    fields:
      campaign-id: "${{ github.run_id }}"
      started-at: "${{ github.event.repository.updated_at }}"
      agent-name: "${{ github.job }}"
    custom-fields:
      - name: "Priority"
        type: "single_select"
        options:
          - "Critical"
          - "High"
          - "Medium"
          - "Low"
        value: "Medium"
        description: "Research priority level"
      - name: "Effort (hours)"
        type: "number"
        value: "4"
        description: "Estimated research effort in hours"
      - name: "Due Date"
        type: "date"
        value: "${{ github.event.repository.updated_at }}"
        description: "Research completion target"
      - name: "Team"
        type: "single_select"
        options:
          - "Research"
          - "Engineering"
          - "Product"
          - "Design"
        value: "Research"
      - name: "Tags"
        type: "text"
        value: "AI, Research, Ethics"
    insights:
      - agent-velocity
      - campaign-progress

safe-outputs:
  create-issue:
    title-prefix: "Research: "
  staged: false

---

# Multi-Agent Research Campaign

You are part of a coordinated research campaign with multiple AI agents working together.

## Your Task

Research one of the following topics and create a comprehensive summary:

**Topics:** {{ inputs.research_topics }}

## Instructions

1. **Select a topic** from the list above (coordinate with other agents if possible)
2. **Research the topic** thoroughly:
   - Key concepts and definitions
   - Current state of the art
   - Main challenges and opportunities
   - Notable researchers and organizations
   - Recent developments (2023-2024)
3. **Create an issue** using the `create-issue` tool with:
   - Title: "Research: [Topic Name]"
   - Body: A well-structured summary with:
     - Overview
     - Key findings
     - Challenges
     - Future directions
     - References (if available)

## Campaign Tracking

This workflow uses a GitHub Project board to track all agents across the campaign:

- **Board:** Research Campaign - ${{ github.run_id }}
- **Your Status:** Will be automatically updated as you work
- **Collaboration:** Check the project board to see what other agents are researching

## Tips

- Be thorough but concise
- Use clear headings and bullet points
- Focus on practical insights
- Include specific examples where relevant
- Cite sources when possible

Good luck! 🚀
