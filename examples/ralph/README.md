# Ralph Loop Example

This directory contains an example of the **Ralph Loop** pattern - an iterative workflow that processes tasks from a PRD (Product Requirements Document) until completion.

## What is Ralph Loop?

The Ralph Loop is a workflow pattern that:
1. Reads a list of tasks from a JSON file
2. Selects the first incomplete task
3. Executes the task using an AI engine
4. Marks the task complete on success
5. Appends learnings to a progress log
6. Repeats until all tasks are done

This pattern is useful for breaking down large projects into manageable, iterative steps that can be completed autonomously by AI agents.

## Files

- **`prd-example.json`** - Example task structure showing how to define tasks with dependencies, priorities, and status tracking
- **`.github/workflows/ralph-loop-basic.md`** - The workflow implementation (located in root `.github/workflows/` directory)

## Usage

### 1. Customize the PRD

Edit `prd-example.json` to define your tasks:

```json
{
  "title": "Your Project Name",
  "description": "Project description",
  "tasks": [
    {
      "id": "task-1",
      "title": "Task name",
      "description": "Detailed task description",
      "status": "incomplete",
      "priority": "high",
      "dependencies": []
    }
  ],
  "progress": []
}
```

**Task Fields:**
- `id` - Unique identifier for the task
- `title` - Short task name
- `description` - Detailed description of what needs to be done
- `status` - Either "incomplete" or "complete"
- `priority` - "high", "medium", or "low"
- `dependencies` - Array of task IDs that must be completed first

### 2. Copy to Your Repository

```bash
# Copy the PRD structure
cp examples/ralph/prd-example.json my-project-prd.json

# The workflow is already available at .github/workflows/ralph-loop-basic.md
```

### 3. Trigger the Workflow

The workflow can be triggered manually:

```bash
gh workflow run ralph-loop-basic.lock.yml
```

Or configure it to run on a schedule in the workflow frontmatter.

### 4. Monitor Progress

The workflow will:
- Read tasks from the PRD file
- Select the first incomplete task with satisfied dependencies
- Execute the task
- Update the task status to "complete"
- Append learnings to `progress.txt`
- Create a pull request with changes

Check the workflow logs and review the PR before merging.

## How It Works

1. **Task Selection**: The workflow finds the first task where:
   - Status is "incomplete"
   - All dependencies are "complete" (or no dependencies exist)

2. **Task Execution**: The selected task is passed to the AI engine with:
   - Task description
   - Repository context
   - Available tools (bash, file operations, etc.)

3. **Quality Checks**: Before marking complete, the workflow runs:
   - Basic linting (if applicable)
   - Unit tests (if they exist)

4. **Progress Tracking**: Each completed task appends to `progress.txt`:
   ```
   [2026-01-22] Completed task-1: Create project structure
   - Set up directory structure with src/, tests/, docs/
   - Created initial configuration files
   - All tests passed
   ```

5. **Pull Request Creation**: Changes are committed via a pull request for human review

6. **Iteration**: After merging the PR, run the workflow again to process the next task

## Customization

### Change the PRD File Path

Edit the workflow frontmatter to use a different file:

```yaml
env:
  PRD_FILE: path/to/your/prd.json
```

### Adjust Quality Checks

Modify the workflow prompt to add custom validation:

```markdown
Before marking the task complete:
1. Run your custom linter
2. Execute specific test suites
3. Verify output quality
```

### Add Tools

Include additional tools in the workflow frontmatter:

```yaml
tools:
  github:
    toolsets: [default]  # For GitHub API access
  web-fetch:             # For external data
  playwright:            # For browser automation
```

## Best Practices

1. **Keep tasks atomic** - Each task should be independently completable
2. **Define clear dependencies** - Ensure tasks build on each other logically
3. **Write descriptive descriptions** - The AI needs clear instructions
4. **Start simple** - Begin with a small PRD to validate the pattern
5. **Review progress regularly** - Monitor the `progress.txt` file and workflow logs

## Example Use Cases

- **Project scaffolding** - Set up new repositories with boilerplate code
- **Documentation generation** - Create docs incrementally as features are built
- **Refactoring projects** - Break down large refactors into manageable steps
- **Content creation** - Generate blog posts, tutorials, or reports step by step
- **Testing campaigns** - Systematically test features and document results

## Limitations

- Tasks must be completable within workflow timeout limits (default: 20 minutes)
- Each task creates a separate PR that must be reviewed and merged
- Complex tasks may require human intervention before merging the PR
- Dependencies are checked but not enforced during execution
- Requires manual workflow runs (or schedule) after each PR merge to continue

## Troubleshooting

**Workflow doesn't select any task:**
- Check that at least one task has `status: "incomplete"`
- Verify dependencies for incomplete tasks are met
- Review workflow logs for task selection output

**Task fails quality checks:**
- Review the error messages in workflow logs
- Fix the issue manually or update the task description
- Re-run the workflow

**PRD file not found:**
- Verify the file path in the workflow environment variables
- Ensure the file is committed to the repository
- Check file permissions

## Contributing

Have improvements to the Ralph Loop pattern? Consider:
- Adding more sophisticated task selection algorithms
- Implementing parallel task execution
- Creating PRD templates for common use cases
- Sharing your customizations with the community

## Related Patterns

- **Campaign Mode** - For running workflows across multiple repositories
- **Scheduled Workflows** - For periodic execution
- **Safe Outputs** - For controlled write operations

## License

This example is part of the GitHub Agentic Workflows project. See the main LICENSE file for details.
