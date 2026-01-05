---
description: Hourly workflow that picks up and executes ready beads from a beads-equipped repository
on:
  schedule:
    - cron: '0 * * * *'  # Every hour
  workflow_dispatch:
permissions:
  contents: write  # Required for bead state updates (reading/writing .beads/ directory)
  issues: read
  pull-requests: read
engine: copilot
strict: false  # Disable strict mode to allow contents: write for beads operations
timeout-minutes: 60
tools:
  github:
    toolsets: [default]
  bash:
    - "*"
  edit:
post-steps:
  - name: Save bead ID for cleanup job
    run: |
      # Save the bead_id to a file that release_bead can read
      BEAD_ID="${{ steps.check_and_claim_bead.outputs.bead_id }}"
      if [ -n "$BEAD_ID" ]; then
        mkdir -p /tmp/gh-aw/bead-context
        echo "$BEAD_ID" > /tmp/gh-aw/bead-context/claimed-bead-id.txt
        echo "Saved claimed bead ID for cleanup: $BEAD_ID"
      fi
  
  - name: Upload bead context
    uses: actions/upload-artifact@v6
    if: always()
    with:
      name: bead-context
      path: /tmp/gh-aw/bead-context/
      if-no-files-found: ignore
steps:
  - name: Check for ready beads and claim one
    id: check_and_claim_bead
    run: |
      # Check if beads are initialized
      if [ ! -d ".beads" ]; then
        echo "bead_id=" >> "$GITHUB_OUTPUT"
        echo "No beads found - repository not initialized with beads"
        exit 0
      fi
      
      # Install beads CLI
      curl -fsSL https://raw.githubusercontent.com/steveyegge/beads/main/scripts/install.sh | bash
      
      # Verify installation
      bd --version
      
      # Get ready beads
      READY_BEADS=$(bd ready --json 2>/dev/null || echo "[]")
      BEAD_COUNT=$(echo "$READY_BEADS" | jq 'length')
      
      echo "Found $BEAD_COUNT ready beads"
      
      if [ "$BEAD_COUNT" -gt 0 ]; then
        # Get the first ready bead
        FIRST_BEAD=$(echo "$READY_BEADS" | jq -r '.[0].id')
        echo "bead_id=$FIRST_BEAD" >> "$GITHUB_OUTPUT"
        echo "Ready to work on bead: $FIRST_BEAD"
        
        # Claim the bead by updating to in_progress
        bd update "$FIRST_BEAD" --status in_progress
        
        # Show bead details
        bd show "$FIRST_BEAD" --json
        
        # Commit the state change
        git config user.name "github-actions[bot]"
        git config user.email "github-actions[bot]@users.noreply.github.com"
        
        if git diff --quiet .beads/; then
          echo "No changes to commit"
        else
          git add .beads/
          git commit -m "Claim bead $FIRST_BEAD as in_progress"
          git push
        fi
      else
        echo "bead_id=" >> "$GITHUB_OUTPUT"
        echo "No ready beads to work on"
        exit 0
      fi
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
        
        - name: Commit bead changes
          run: |
            git config user.name "github-actions[bot]"
            git config user.email "github-actions[bot]@users.noreply.github.com"
            
            # Commit changes to .beads/ if any
            if git diff --quiet .beads/; then
              echo "No bead changes to commit"
            else
              git add .beads/
              git commit -m "Update bead ${{ inputs.bead_id }} to ${{ inputs.state }}"
              git push
            fi
jobs:
  release_bead:
    needs: [agent]
    if: always()
    runs-on: ubuntu-latest
    permissions:
      contents: write
    steps:
      - name: Download bead context
        uses: actions/download-artifact@v6
        continue-on-error: true
        with:
          name: bead-context
          path: /tmp/gh-aw/bead-context/
      
      - name: Check if bead was claimed
        id: check_bead_id
        run: |
          # Read the claimed bead ID from the artifact
          if [ -f "/tmp/gh-aw/bead-context/claimed-bead-id.txt" ]; then
            BEAD_ID=$(cat /tmp/gh-aw/bead-context/claimed-bead-id.txt)
            echo "Bead claimed by workflow: $BEAD_ID"
            echo "should_release=true" >> "$GITHUB_OUTPUT"
            echo "bead_id=$BEAD_ID" >> "$GITHUB_OUTPUT"
          else
            echo "No bead was claimed by this workflow - skipping release"
            echo "should_release=false" >> "$GITHUB_OUTPUT"
          fi
      
      - name: Checkout repository
        if: steps.check_bead_id.outputs.should_release == 'true'
        uses: actions/checkout@v5
        with:
          token: ${{ secrets.GITHUB_TOKEN }}
          fetch-depth: 0
      
      - name: Install beads
        if: steps.check_bead_id.outputs.should_release == 'true'
        run: |
          # Install beads CLI
          curl -fsSL https://raw.githubusercontent.com/steveyegge/beads/main/scripts/install.sh | bash
          
          # Verify installation
          bd --version
      
      - name: Release the claimed bead if still in progress
        if: steps.check_bead_id.outputs.should_release == 'true'
        env:
          BEAD_ID: ${{ steps.check_bead_id.outputs.bead_id }}
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
            
            # Commit the state change
            git config user.name "github-actions[bot]"
            git config user.email "github-actions[bot]@users.noreply.github.com"
            
            if git diff --quiet .beads/; then
              echo "No changes to commit"
            else
              git add .beads/
              git commit -m "Release bead $BEAD_ID back to open state"
              git push
            fi
          else
            echo "Bead is no longer in progress (status: $BEAD_STATUS) - no need to release"
          fi
---

# Beads Worker

You are an automated beads worker that processes ready tasks from a beads-equipped repository.

## Context

- **Repository**: ${{ github.repository }}
- **Bead ID**: ${{ steps.check_and_claim_bead.outputs.bead_id }}
- **Run URL**: ${{ github.server_url }}/${{ github.repository }}/actions/runs/${{ github.run_id }}

## Your Mission

1. **Get bead details**: Use `bd show ${{ steps.check_and_claim_bead.outputs.bead_id }} --json` to get full details about the bead
2. **Understand the task**: Read the bead title, description, and any related context
3. **Check dependencies**: Review dependency tree with `bd dep tree ${{ steps.check_and_claim_bead.outputs.bead_id }}`
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
   - `bead_id`: "${{ steps.check_and_claim_bead.outputs.bead_id }}"
   - `state`: "closed"
   - `reason`: Brief description of what was completed
4. If you cannot complete the task, call `bead-update-state` with:
   - `bead_id`: "${{ steps.check_and_claim_bead.outputs.bead_id }}"
   - `state`: "open"
   - `reason`: Explanation of why the task couldn't be completed

## Important Notes

- The bead has been claimed as `in_progress` in the activation job
- If you don't explicitly close or reopen the bead, the conclusion job will automatically release it back to `open` state
- Always provide a clear reason when updating bead state
- Document any significant work in a GitHub issue for tracking

Begin by examining the bead and executing the work!
