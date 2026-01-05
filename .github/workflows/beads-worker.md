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
if: needs.bead.outputs.id != ''
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
  create-pull-request:
    title-prefix: "[beads] "
    labels: [automation, beads]
    draft: true
  push-to-pull-request-branch:
    target: "*"
  jobs:
    bead-update-state:
      description: "Update bead state (completed, failed, or released)"
      runs-on: ubuntu-latest
      output: "Bead state updated successfully"
      permissions:
        contents: write
      inputs:
        state:
          description: "The new state (open, in_progress, closed)"
          required: true
          type: string
          options:
            - open
            - in_progress
            - closed
        reason:
          description: "Reason for the state change"
          required: false
          type: string
      steps:
        - name: Setup git configuration
          run: |
            git config --global user.name "github-actions[bot]"
            git config --global user.email "github-actions[bot]@users.noreply.github.com"
        
        - name: Checkout .beads folder
          uses: actions/checkout@v4
          with:
            ref: beads-sync
            sparse-checkout: |
              .beads
            persist-credentials: true
        
        - name: Sync with beads
          env:
            GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
          run: |
            echo "=== Syncing beads data ==="
            # Install beads CLI
            curl -fsSL https://raw.githubusercontent.com/steveyegge/beads/main/scripts/install.sh | bash
            
            # Verify installation
            bd --version
            
            # Sync beads data from repository
            bd --no-db sync
            echo "✓ Beads data synced"
        
        - name: Update bead state
          env:
            BEAD_ID: ${{ needs.bead.outputs.id }}
            STATE: ${{ inputs.state }}
            REASON: ${{ inputs.reason }}
          run: |
            echo "=== Starting bead state update ==="
            echo "Bead ID: $BEAD_ID (claimed bead)"
            echo "New state: $STATE"
            echo "Reason: $REASON"
            
            # Update bead state
            echo "Updating bead state..."
            if [ -n "$REASON" ]; then
              bd update "$BEAD_ID" --status "$STATE" --comment "$REASON"
            else
              bd update "$BEAD_ID" --status "$STATE"
            fi
            echo "✓ Bead state updated successfully"
            
            # Show updated bead
            echo "Updated bead details:"
            bd show "$BEAD_ID" --json
            echo "=== Bead state update completed ==="
        
        - name: Sync bead changes
          run: |
            echo "=== Starting bead sync ==="
            echo "Syncing changes to repository..."
            bd --no-db sync
            echo "✓ Sync completed successfully"
            echo "=== Bead sync completed ==="
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
      - name: Setup git configuration
        run: |
          git config --global user.name "github-actions[bot]"
          git config --global user.email "github-actions[bot]@users.noreply.github.com"
      
      - name: Checkout .beads folder
        uses: actions/checkout@v4
        with:
          ref: beads-sync
          sparse-checkout: |
            .beads
          persist-credentials: true
      
      - name: Install beads CLI
        run: |
          # Install beads CLI
          curl -fsSL https://raw.githubusercontent.com/steveyegge/beads/main/scripts/install.sh | bash
          
          # Verify installation
          bd --version
      
      - name: Claim ready bead
        id: claim_bead
        env:
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
        run: |
          echo "=== Starting bead claim process ==="
          echo "Timestamp: $(date -u +"%Y-%m-%dT%H:%M:%SZ")"
          
          # Sync beads data from repository
          echo "Syncing beads data..."
          bd --no-db sync
          echo "✓ Beads data synced"
          
          # Check if beads are initialized
          echo "Checking for .beads directory..."
          if [ ! -d ".beads" ]; then
            echo "id=" >> "$GITHUB_OUTPUT"
            echo "title=" >> "$GITHUB_OUTPUT"
            echo "description=" >> "$GITHUB_OUTPUT"
            echo "status=" >> "$GITHUB_OUTPUT"
            echo "⚠️  No beads found - repository not initialized with beads"
            exit 0
          fi
          echo "✓ .beads directory found"
          
          # Get ready beads (without daemon)
          echo "Fetching ready beads..."
          READY_BEADS=$(bd ready --json --no-daemon 2>/dev/null || echo "[]")
          BEAD_COUNT=$(echo "$READY_BEADS" | jq 'length')
          
          echo "✓ Found $BEAD_COUNT ready beads"
          
          if [ "$BEAD_COUNT" -gt 0 ]; then
            # Get the first ready bead
            echo "Processing first ready bead..."
            BEAD_ID=$(echo "$READY_BEADS" | jq -r '.[0].id')
            BEAD_TITLE=$(echo "$READY_BEADS" | jq -r '.[0].title // ""')
            BEAD_DESC=$(echo "$READY_BEADS" | jq -r '.[0].description // ""')
            
            echo "id=$BEAD_ID" >> "$GITHUB_OUTPUT"
            echo "title=$BEAD_TITLE" >> "$GITHUB_OUTPUT"
            echo "description=$BEAD_DESC" >> "$GITHUB_OUTPUT"
            echo "status=claimed" >> "$GITHUB_OUTPUT"
            
            echo "✓ Bead selected:"
            echo "  - ID: $BEAD_ID"
            echo "  - Title: $BEAD_TITLE"
            echo "  - Description: $BEAD_DESC"
            
            # Claim the bead by updating to in_progress
            echo "Claiming bead (updating to in_progress)..."
            bd update "$BEAD_ID" --status in_progress
            echo "✓ Bead claimed successfully"
            
            # Show bead details
            echo "Bead details:"
            bd show "$BEAD_ID" --json
          else
            echo "id=" >> "$GITHUB_OUTPUT"
            echo "title=" >> "$GITHUB_OUTPUT"
            echo "description=" >> "$GITHUB_OUTPUT"
            echo "status=none" >> "$GITHUB_OUTPUT"
            echo "ℹ️  No ready beads to work on"
          fi
          
          echo "=== Bead claim process completed ==="
      
      - name: Sync bead changes
        if: steps.claim_bead.outputs.id != ''
        run: |
          echo "=== Starting bead sync ==="
          echo "Syncing changes to repository..."
          
          bd --no-db sync
          echo "✓ Sync completed successfully"
          echo "=== Bead sync completed ==="
  
  release_bead:
    needs: [bead, agent]
    if: always() && needs.bead.outputs.id != ''
    runs-on: ubuntu-latest
    permissions:
      contents: write
    steps:
      - name: Setup git configuration
        run: |
          git config --global user.name "github-actions[bot]"
          git config --global user.email "github-actions[bot]@users.noreply.github.com"
      
      - name: Checkout .beads folder
        uses: actions/checkout@v4
        with:
          ref: beads-sync
          sparse-checkout: |
            .beads
          persist-credentials: true
      
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
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
        run: |
          echo "=== Starting bead release process ==="
          echo "Claimed bead ID: $BEAD_ID"
          echo "Agent result: $AGENT_RESULT"
          
          # Sync beads data from repository
          echo "Syncing beads data..."
          bd --no-db sync
          echo "✓ Beads data synced"
          
          # Check if the claimed bead is still in_progress
          echo "Checking bead status..."
          BEAD_STATUS=$(bd show "$BEAD_ID" --json 2>/dev/null | jq -r '.status')
          
          echo "Current bead status: $BEAD_STATUS"
          
          if [ "$BEAD_STATUS" = "in_progress" ]; then
            echo "⚠️  Bead is still in progress - releasing it back to open state"
            bd update "$BEAD_ID" --status open --comment "Released by workflow (agent result: $AGENT_RESULT)"
            echo "✓ Bead released successfully"
            
            # Sync changes
            echo "Syncing changes to repository..."
            bd --no-db sync
            echo "✓ Sync completed successfully"
          else
            echo "ℹ️  Bead is no longer in progress (status: $BEAD_STATUS) - no need to release"
          fi
          
          echo "=== Bead release process completed ==="
---

# Beads Worker

You are an automated beads worker that processes ready tasks from a beads-equipped repository.

<!--
BEAD STATE MACHINE:

States:
  - open: Bead is ready to be claimed and worked on
  - in_progress: Bead has been claimed and is being worked on
  - closed: Bead work is complete

Transitions:
  open -> in_progress: When bead job claims a ready bead
  in_progress -> closed: When agent successfully completes the work (via bead-update-state)
  in_progress -> open: When agent cannot complete work (via bead-update-state) OR when release_bead job runs (timeout/failure)
  
Workflow:
  1. bead job: Finds ready beads (state=open) and claims first one (state=in_progress)
  2. agent job: Works on the bead and updates state via bead-update-state tool
     - Success: in_progress -> closed
     - Cannot complete: in_progress -> open
  3. release_bead job (if: always()): If bead is still in_progress, releases it back to open
-->

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

- **bead-update-state**: Update bead state after completing work (automatically uses the claimed bead)
  - Use `state: closed` with `reason: "Task completed: [description]"` when work is done successfully
  - Use `state: open` with `reason: "Failed: [reason]"` if you cannot complete the work

- **create_issue**: Create GitHub issues for tracking work or reporting findings
  - Use to document bugs, feature requests, or work items discovered while executing a bead
  - Issues will be automatically labeled with `automation` and `beads` and prefixed with `[beads]`

- **create_pull_request**: Create pull requests with code changes
  - Use when the bead task involves code modifications that should be reviewed
  - PRs will be created as drafts by default, labeled with `automation` and `beads`, and prefixed with `[beads]`

- **push_to_pull_request_branch**: Push additional changes to an existing pull request
  - Use to update or refine changes in a PR that was previously created
  - Can target any open pull request in the repository

## Execution Guidelines

1. Start by examining the bead details thoroughly
2. If the task involves code changes:
   - Make the necessary changes
   - Test your changes if possible
   - Consider creating a pull request using `create_pull_request` for review
3. If you discover bugs or work items while executing:
   - Use `create_issue` to document them for future action
4. If the task is complete, call `bead-update-state` with:
   - `state`: "closed"
   - `reason`: Brief description of what was completed
5. If you cannot complete the task, call `bead-update-state` with:
   - `state`: "open"
   - `reason`: Explanation of why the task couldn't be completed

## Important Notes

- The bead has been claimed as `in_progress` in the bead job
- If you don't explicitly close or reopen the bead, the release_bead job will automatically release it back to `open` state
- Always provide a clear reason when updating bead state
- Use `create_issue` to document any significant work, bugs, or follow-up items
- Use `create_pull_request` when code changes require review before merging
- Use `push_to_pull_request_branch` to update existing PRs with additional changes

Begin by examining the bead and executing the work!
