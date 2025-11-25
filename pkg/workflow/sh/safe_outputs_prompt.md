<safe-outputs>
<description>GitHub API Access Instructions</description>

<important>
The gh (GitHub CLI) command is NOT authenticated in this environment. Do NOT use gh commands for GitHub API operations.
</important>

<instructions>
To interact with GitHub (create issues, discussions, comments, pull requests, etc.), use the safe output tools provided by the safeoutputs MCP server instead of the gh CLI.
</instructions>

<available-tools>
The safeoutputs MCP server provides these tools:
- create_issue - Create GitHub issues
- create_discussion - Create GitHub discussions
- add_comment - Add comments to issues, PRs, or discussions
- create_pull_request - Create pull requests
- create_pull_request_review_comment - Add review comments on PR code
- add_labels - Add labels to issues or PRs
- add_reviewer - Add reviewers to pull requests
- update_issue - Update issue status, title, or body
- close_issue - Close issues with a comment
- close_discussion - Close discussions with a comment
- close_pull_request - Close PRs without merging
- push_to_pull_request_branch - Push changes to PR branches
- assign_milestone - Assign issues to milestones
- assign_to_agent - Assign GitHub Copilot agent to issues
- create_agent_task - Create GitHub Copilot agent tasks
- create_code_scanning_alert - Create code scanning alerts
- upload_asset - Publish files as URL-addressable assets
- update_release - Update release descriptions
- noop - Log completion messages for transparency
- missing_tool - Report missing tools or functionality
</available-tools>

<reminder>Use these MCP tools instead of gh CLI commands for all GitHub API operations.</reminder>
</safe-outputs>
