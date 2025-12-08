# AI Moderator Workflow

An AI-powered GitHub Agentic Workflow that automatically detects spam in issues and comments, inspired by [github/ai-moderator](https://github.com/github/ai-moderator).

## Features

- **Automatic Spam Detection**: Detects promotional content, scams, and irrelevant posts
- **Link Spam Detection**: Identifies suspicious URLs and shortened links
- **AI-Generated Content Detection**: Recognizes artificially generated content patterns
- **Automated Actions**: 
  - Adds labels (`spam`, `link-spam`, `ai-generated`) to flagged issues
  - Minimizes detected spam comments
- **Multiple Trigger Support**: Works on new issues, issue comments, and PR review comments

## How It Works

This workflow uses GitHub Copilot's AI capabilities to analyze content posted to your repository. When triggered by:
- A new issue being opened
- A new comment on an issue
- A new review comment on a pull request

The AI agent:
1. Fetches the content to analyze
2. Runs three detection analyses (general spam, link spam, AI-generated)
3. Takes appropriate action based on findings:
   - For issues: Adds relevant labels
   - For comments: Minimizes the comment and adds labels to the parent issue/PR

## Detection Criteria

### Generic Spam
- Promotional content or advertisements
- Cryptocurrency or financial scams
- Repetitive text patterns
- Low-quality or nonsensical content
- Requests for personal information

### Link Spam
- Multiple unrelated links
- Short URL services (bit.ly, tinyurl, etc.)
- Links to promotional, gambling, or adult content
- Suspicious or newly registered domains
- Links to executables or suspicious downloads

### AI-Generated Content
- Overly formal responses in casual contexts
- Perfect grammar in informal settings
- Excessive emoji usage in technical discussions
- Generic enthusiasm without substance
- Perfectly structured responses lacking natural flow

## Configuration

The workflow is configured in `.github/workflows/ai-moderator.md` with the following settings:

```yaml
timeout-minutes: 10
on:
  issues:
    types: [opened]
  issue_comment:
    types: [created]
  pull_request_review_comment:
    types: [created]
permissions:
  models: read
  contents: read
  issues: read
  pull-requests: read
```

### Labels

The workflow uses three labels:
- `spam` - Generic spam content
- `link-spam` - Suspicious or promotional links
- `ai-generated` - AI-generated content

**Important**: These labels must exist in your repository before the workflow can add them. 

**To create the labels**, you can:
1. **Via GitHub UI**: Go to your repository → Issues → Labels → New label
2. **Via GitHub CLI**:
   ```bash
   gh label create spam --description "Generic spam content" --color d73a4a
   gh label create link-spam --description "Suspicious or promotional links" --color d73a4a
   gh label create ai-generated --description "AI-generated content" --color fbca04
   ```
3. **Via API**: Use the GitHub REST API to create labels programmatically

If the labels don't exist, the workflow will fail when trying to add them to issues.

### Custom Safe Output: Minimize Comment

The workflow includes a custom safe output job that uses GitHub's GraphQL API to minimize comments detected as spam. This job requires:
- `comment_id`: The numeric ID of the comment
- `comment_node_id`: The GraphQL node ID of the comment
- `classifier`: The reason for minimizing (defaults to "SPAM")

## Security

The workflow follows security best practices:
- **Read-only main job**: The AI analysis job only has read permissions
- **Safe outputs**: Write operations (labeling, minimizing) are handled by separate jobs with explicit permissions
- **Strict mode**: Enforces security constraints to prevent unauthorized actions

## Comparison with github/ai-moderator

| Feature | github/ai-moderator | gh-aw AI Moderator |
|---------|---------------------|-------------------|
| Spam Detection | ✅ | ✅ |
| Link Spam Detection | ✅ | ✅ |
| AI Content Detection | ✅ | ✅ |
| Label Application | ✅ | ✅ |
| Comment Minimization | ✅ | ✅ |
| Custom Prompts | ✅ File-based | ✅ Inline in workflow |
| Dry-run Mode | ✅ | 🔜 (future enhancement) |
| Configuration | Action inputs | Workflow frontmatter |
| AI Engine | GitHub Models | GitHub Copilot |

## Customization

To customize the detection behavior:

1. **Edit the workflow**: Modify `.github/workflows/ai-moderator.md`
2. **Adjust detection criteria**: Update the prompt sections for each detection type
3. **Change labels**: Modify the `allowed` list in `safe-outputs.add-labels`
4. **Add/remove triggers**: Update the `on:` section
5. **Recompile**: Run `gh aw compile ai-moderator` to generate the updated `.lock.yml` file

## Example Workflow Run

1. A user opens a new issue with spam content
2. The workflow is triggered automatically
3. The AI agent fetches the issue content
4. Analyzes the content for spam, link spam, and AI-generated patterns
5. Detects spam indicators with high confidence
6. Adds the `spam` label to the issue
7. Logs the detection reasoning for transparency

## Limitations

- Requires GitHub Copilot access (via `engine: copilot` and appropriate permissions)
- Labels must be pre-created in the repository
- Conservative detection to minimize false positives
- May not catch sophisticated or evolving spam patterns

## Contributing

To improve the detection accuracy:
1. Review false positives/negatives in your workflow runs
2. Adjust the detection criteria in the workflow prompt
3. Test with `gh aw trial` before deploying
4. Submit feedback or improvements via pull request

## License

This workflow is part of the gh-aw project and follows the same license terms.
