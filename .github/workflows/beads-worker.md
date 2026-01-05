---
description: Hourly workflow that picks up and executes ready beads from a beads-equipped repository
on:
  schedule: hourly
  workflow_dispatch:
permissions:
  contents: read
  issues: read
  pull-requests: read
engine: copilot
strict: true
timeout-minutes: 60
tools:
  github:
    toolsets: [default]
  bash:
    - "*"
  edit:
safe-outputs:
  create-issue:
    title-prefix: "[beads] "
    labels: [automation, beads]
    max: 3
  jobs:
    bead-update-state:
      description: "Update bead state (completed, failed, or released)"
      runs-on: ubuntu-latest
      output: "Bead state updated successfully"
      permissions:
        contents: write
      inputs:
        bead_id:
          description: "The bead ID to update"
          required: true
          type: string
        state:
          description: "The new state (open, in_progress, closed)"
          required: true
          type: string
        reason:
          description: "Reason for the state change"
          required: false
          type: string
      steps:
        - name: Checkout repository
          uses: actions/checkout@v5
          with:
            token: ${{ secrets.GITHUB_TOKEN }}
            fetch-depth: 0
        
        - name: Install beads
          run: |
            # Install beads CLI
            curl -fsSL https://raw.githubusercontent.com/steveyegge/beads/main/scripts/install.sh | bash
            
            # Verify installation
            bd --version
        
        - name: Update bead state
          env:
            BEAD_ID: ${{ inputs.bead_id }}
            STATE: ${{ inputs.state }}
            REASON: ${{ inputs.reason }}
          run: |
            # Update bead state
            if [ -n "$REASON" ]; then
              bd update "$BEAD_ID" --status "$STATE" --comment "$REASON"
            else
              bd update "$BEAD_ID" --status "$STATE"
            fi
            
            # Show updated bead
            bd show "$BEAD_ID" --json
        
        - name: Sync bead changes
          run: |
            # Sync changes to repository
            bd sync
jobs:
  bead:
    needs: activation
    runs-on: ubuntu-latest
    permissions:
      contents: write
    outputs:
      id: ${{ steps.claim_bead.outputs.id }}
      title: ${{ steps.claim_bead.outputs.title }}
      description: ${{ steps.claim_bead.outputs.description }}
      status: ${{ steps.claim_bead.outputs.status }}
    steps:
      - name: Checkout repository
        uses: actions/checkout@v5
        with:
          token: ${{ secrets.GITHUB_TOKEN }}
          fetch-depth: 0
      
      - name: Install beads CLI
        run: |
          # Install beads CLI
          curl -fsSL https://raw.githubusercontent.com/steveyegge/beads/main/scripts/install.sh | bash
          
          # Verify installation
          bd --version
      
      - name: Claim ready bead
        id: claim_bead
        run: |
          # Check if beads are initialized
          if [ ! -d ".beads" ]; then
            echo "id=" >> "$GITHUB_OUTPUT"
            echo "title=" >> "$GITHUB_OUTPUT"
            echo "description=" >> "$GITHUB_OUTPUT"
            echo "status=" >> "$GITHUB_OUTPUT"
            echo "No beads found - repository not initialized with beads"
            exit 0
          fi
          
          # Get ready beads (without daemon)
          READY_BEADS=$(bd ready --json --no-daemon 2>/dev/null || echo "[]")
          BEAD_COUNT=$(echo "$READY_BEADS" | jq 'length')
          
          echo "Found $BEAD_COUNT ready beads"
          
          if [ "$BEAD_COUNT" -gt 0 ]; then
            # Get the first ready bead
            BEAD_ID=$(echo "$READY_BEADS" | jq -r '.[0].id')
            BEAD_TITLE=$(echo "$READY_BEADS" | jq -r '.[0].title // ""')
            BEAD_DESC=$(echo "$READY_BEADS" | jq -r '.[0].description // ""')
            
            echo "id=$BEAD_ID" >> "$GITHUB_OUTPUT"
            echo "title=$BEAD_TITLE" >> "$GITHUB_OUTPUT"
            echo "description=$BEAD_DESC" >> "$GITHUB_OUTPUT"
            echo "status=claimed" >> "$GITHUB_OUTPUT"
            
            echo "Ready to work on bead: $BEAD_ID"
            echo "Title: $BEAD_TITLE"
            
            # Claim the bead by updating to in_progress
            bd update "$BEAD_ID" --status in_progress
            
            # Show bead details
            bd show "$BEAD_ID" --json
          else
            echo "id=" >> "$GITHUB_OUTPUT"
            echo "title=" >> "$GITHUB_OUTPUT"
            echo "description=" >> "$GITHUB_OUTPUT"
            echo "status=none" >> "$GITHUB_OUTPUT"
            echo "No ready beads to work on"
          fi
      
      - name: Sync bead changes
        if: steps.claim_bead.outputs.id != ''
        run: |
          # Sync changes to repository
          bd sync
  
  release_bead:
    needs: [bead, agent]
    if: always() && needs.bead.outputs.id != ''
    runs-on: ubuntu-latest
    permissions:
      contents: write
    steps:
      - name: Checkout repository
        uses: actions/checkout@v5
        with:
          token: ${{ secrets.GITHUB_TOKEN }}
          fetch-depth: 0
      
      - name: Install beads CLI
        run: |
          # Install beads CLI
          curl -fsSL https://raw.githubusercontent.com/steveyegge/beads/main/scripts/install.sh | bash
          
          # Verify installation
          bd --version
      
      - name: Release the claimed bead if still in progress
        env:
          BEAD_ID: ${{ needs.bead.outputs.id }}
          AGENT_RESULT: ${{ needs.agent.result }}
        run: |
          # Check if the claimed bead is still in_progress
          BEAD_STATUS=$(bd show "$BEAD_ID" --json 2>/dev/null | jq -r '.status')
          
          echo "Claimed bead: $BEAD_ID"
          echo "Current status: $BEAD_STATUS"
          echo "Agent result: $AGENT_RESULT"
          
          if [ "$BEAD_STATUS" = "in_progress" ]; then
            echo "Bead is still in progress - releasing it back to open state"
            bd update "$BEAD_ID" --status open --comment "Released by workflow (agent result: $AGENT_RESULT)"
            
            # Sync changes
            bd sync
          else
            echo "Bead is no longer in progress (status: $BEAD_STATUS) - no need to release"
          fi
---

# Beads Worker

You are an automated beads worker that processes ready tasks from a beads-equipped repository.

## Context

- **Repository**: ${{ github.repository }}
- **Bead ID**: ${{ needs.bead.outputs.id }}
- **Bead Title**: ${{ needs.bead.outputs.title }}
- **Bead Description**: ${{ needs.bead.outputs.description }}
- **Run URL**: ${{ github.server_url }}/${{ github.repository }}/actions/runs/${{ github.run_id }}

## Your Mission

1. **Get bead details**: Use `bd show ${{ needs.bead.outputs.id }} --json` to get full details about the bead
2. **Understand the task**: Read the bead title, description, and any related context
3. **Check dependencies**: Review dependency tree with `bd dep tree ${{ needs.bead.outputs.id }}`
4. **Execute the work**: Complete the task described in the bead
5. **Report completion**: Use the `bead-update-state` tool to update the bead state

## Tools Available

- **bead-update-state**: Update bead state after completing work
  - Use `state: closed` with `reason: "Task completed: [description]"` when work is done successfully
  - Use `state: open` with `reason: "Failed: [reason]"` if you cannot complete the work

## Execution Guidelines

1. Start by examining the bead details thoroughly
2. If the task involves code changes:
   - Make the necessary changes
   - Test your changes if possible
   - Create an issue documenting what was done (use create-issue safe output)
3. If the task is complete, call `bead-update-state` with:
   - `bead_id`: "${{ needs.bead.outputs.id }}"
   - `state`: "closed"
   - `reason`: Brief description of what was completed
4. If you cannot complete the task, call `bead-update-state` with:
   - `bead_id`: "${{ needs.bead.outputs.id }}"
   - `state`: "open"
   - `reason`: Explanation of why the task couldn't be completed

## Important Notes

- The bead has been claimed as `in_progress` in the bead job
- If you don't explicitly close or reopen the bead, the release_bead job will automatically release it back to `open` state
- Always provide a clear reason when updating bead state
- Document any significant work in a GitHub issue for tracking

Begin by examining the bead and executing the work!
