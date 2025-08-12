---
on:
  issues:
    types: [opened]

permissions:
  issues: write
  contents: read

tools:
  github:
    allowed: 
      - get_issue
      - search_issues
      - add_issue_comment
      - update_issue

engine: gemini
timeout_minutes: 10
---

# Issue Duplicate Detection with Gemini

Analyze the newly opened issue #${{ github.event.issue.number }} to detect potential duplicates and provide helpful context.

## Your Task

1. **Read the new issue**: Use the GitHub tools to get comprehensive details of issue #${{ github.event.issue.number }} including title, body, labels, and any existing comments.

2. **Search for similar issues**: Systematically search the repository for existing issues that might be duplicates by:
   - Looking for issues with similar titles or keywords
   - Searching for issues with related error messages or symptoms  
   - Checking both open and closed issues across different timeframes
   - Using multiple search strategies to ensure comprehensive coverage

3. **Analyze similarities**: For each potentially similar issue found:
   - Compare the problem descriptions and symptoms
   - Look for matching error messages, stack traces, or technical details
   - Consider if they describe the same underlying issue or root cause
   - Check if they request the same feature enhancement or bug fix
   - Assess the quality and completeness of each issue's information

4. **Take action**: Based on your analysis, add a helpful comment:
   - **If clear duplicates found**: Explain the duplication with specific reasoning, mention the duplicate issue numbers with links, and suggest appropriate action
   - **If related but not duplicate issues found**: Reference related issues for valuable context and cross-linking
   - **If no duplicates found**: Acknowledge the new issue, confirm it appears unique, and provide any helpful initial observations

## Response Guidelines

- Start your comment with "🔍 **Duplicate Detection Analysis**"
- Provide clear, specific reasoning for your assessments
- Include issue numbers with links when referencing other issues (e.g., "Similar to #123")
- Use a helpful, professional tone that welcomes the contributor
- Keep responses well-structured but appropriately detailed
- Focus on being genuinely helpful to both the issue reporter and maintainers

## Repository Context

**Repository**: ${{ github.repository }}  
**New Issue**: #${{ github.event.issue.number }}  
**Issue Title**: "${{ github.event.issue.title }}"  
**Opened by**: ${{ github.actor }}

@include shared/issue-reader.md

@include shared/tool-refused.md

@include shared/include-link.md

@include shared/job-summary.md

@include shared/xpia.md