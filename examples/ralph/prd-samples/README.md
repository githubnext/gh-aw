# Ralph PRD Examples

This directory contains example Product Requirements Documents (PRD) for use with the Ralph Loop workflow.

## About Ralph Loop

The Ralph Loop workflow uses structured PRD files in JSON format to generate user stories and track development progress. The PRD format breaks down features into small, atomic user stories with clear acceptance criteria.

## Files

- **`feature-example.md`**: A sample feature description for a task management system
  - Shows the format and level of detail needed for PRD generation
  - Includes requirements, technical context, and expected outcomes

- **`prd-output.json`**: The generated PRD in Ralph Loop format
  - Demonstrates the JSON structure required for Ralph Loop
  - Contains 8 user stories broken down from the feature description
  - Each story includes title, description, acceptance criteria, and status

## Usage

### Generating a PRD

Use the Ralph PRD Generator workflow to create a structured PRD from a feature description:

1. Go to Actions → Ralph PRD Generator
2. Click "Run workflow"
3. Enter your feature description
4. Optionally specify a branch name
5. The workflow will generate a `prd.json` file and create a PR

### PRD Structure

```json
{
  "feature": "Brief feature name",
  "branchName": "kebab-case-branch-name",
  "userStories": [
    {
      "id": 1,
      "title": "Action-oriented story title",
      "description": "Detailed description with context",
      "acceptanceCriteria": [
        "Specific, testable criterion 1",
        "Specific, testable criterion 2"
      ],
      "passes": false
    }
  ]
}
```

## Best Practices

### For User Stories

1. **Keep them small**: Each story should be completable in 1-2 hours
2. **Single focus**: One story = one capability or change
3. **Independent**: Minimize dependencies between stories
4. **Testable**: Clear acceptance criteria that can be verified

### For Acceptance Criteria

1. **Specific**: Avoid vague language like "should work well"
2. **Testable**: Can be verified with concrete tests
3. **Behavioral**: Focus on what should happen, not how
4. **Complete**: Cover success cases, error cases, and edge cases

## Example Feature Types

The PRD generator works well for:

- UI components and features
- API endpoints and services
- Data models and schemas
- Integrations and workflows
- Configuration and setup tasks

## Tips

- Provide enough context in your feature description
- Include technical constraints or preferences
- Mention any existing code or patterns to follow
- Specify the expected outcome clearly
- Break down complex features into multiple PRDs if needed
