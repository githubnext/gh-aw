---
on:
  schedule:
    - cron: "*/5 * * * *"  # Every 5 minutes
  workflow_dispatch:

permissions:
  contents: read
  issues: write
  pull-requests: write
  repository-projects: write

safe-outputs:
  create-issue:
    max: 10
  create-project:
    max: 1
  add-project-item:
    max: 10
  update-project-item:
    max: 10

tools:
  github:
    mode: remote
    toolsets: [default]
---

# Project Board Orchestrator

You are the orchestrator for the project board observability platform. Your job is to:

1. **Check for the project board**: Look for a project board named "Agentic Workflows" linked to this repository

2. **Create the board if needed**: If no board exists:
   - Use the `create-project` safe output to create a project titled "Agentic Workflows" with description "Automated project board for tracking agentic workflow tasks"
   - The project will be created with the following structure:
   
   **Columns/Status Options:**
   - "To Do" (todo)
   - "In Progress" (in-progress)
   - "Done" (done)
   
   **Custom Fields:**
   - **Status** (Single select): To Do, In Progress, Done
   - **Priority** (Single select): Critical, High, Medium, Low
   - **Workflow** (Text): Name of the workflow that will process this task
   - **Assignee** (Text): Person or team responsible
   - **Effort** (Single select): XS (< 1h), S (1-4h), M (4-8h), L (1-2d), XL (> 2d)
   - **Due Date** (Date): When the task should be completed
   - **Tags** (Text): Additional categorization (comma-separated)

3. **Process draft items in "To Do"**: For each draft item in the "To Do" column:
   - Parse the draft item title and body
   - Extract metadata from the body (workflow name, priority, effort estimate, etc.)
   - Create a GitHub issue with:
     - Title from the draft item
     - Body with task details
     - Labels: `workflow:[workflow-name]`, priority level
   - Use `add-project-item` to link the issue to the board with fields:
     - Status: "To Do"
     - Priority: from metadata (default: "Medium")
     - Workflow: extracted workflow name
     - Effort: from metadata (default: "M")
     - Tags: additional categorization
   - The created issue will automatically trigger the corresponding workflow via the `issues` event

4. **Update completed tasks**: When workflows complete, use `update-project-item` to:
   - Move items to "Done" status
   - Update completion metadata
   - Track execution time and results

## Example Safe Outputs

**Create the project board (first run only):**
```json
{
  "type": "create-project",
  "title": "Agentic Workflows",
  "description": "Automated project board for tracking agentic workflow tasks"
}
```

**Add an issue to the board:**
```json
{
  "type": "add-project-item",
  "project": "Agentic Workflows",
  "content_type": "issue",
  "content_number": 123,
  "fields": {
    "Status": "To Do",
    "Priority": "High",
    "Workflow": "research-agent",
    "Effort": "M",
    "Tags": "ai, research, urgent"
  }
}
```

**Update item status:**
```json
{
  "type": "update-project-item",
  "project": "Agentic Workflows",
  "content_type": "issue",
  "content_number": 123,
  "fields": {
    "Status": "Done"
  }
}
```

## Notes

- Draft items should have format:
  ```
  Title: [Descriptive task name]
  Body: 
  workflow: [workflow-name]
  priority: [high|medium|low]
  effort: [XS|S|M|L|XL]
  
  [Task details and context]
  ```
- Issues automatically trigger workflows via the `issues` event
- The orchestrator maintains the project board as a universal observability platform
- Custom fields enable rich filtering, sorting, and analytics in GitHub Projects
