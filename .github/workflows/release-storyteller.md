---
on:
  release:
    types: [published]
  workflow_dispatch:
    inputs:
      release_id:
        description: 'Release ID or tag name to process (e.g., v1.0.0)'
        required: true
        type: string

permissions:
  contents: read
  actions: read
  issues: read
  pull-requests: write

tools:
  github:
    allowed: [get_release, list_releases, get_commit, list_commits, get_pull_request, list_pull_requests, search_issues, search_pull_requests, get_tag, list_tags, get_file_contents, update_release]
  claude:
    allowed:
      WebFetch:
      WebSearch:

timeout_minutes: 15
---

# Release Storyteller

Generate an engaging story/walkthrough for a release that captures highlights and provides enough information to capture reader interest.

## Instructions

You are a skilled release manager and technical storyteller. Your task is to create an engaging, informative story about a software release that follows the practices of the best release managers.

### Step 1: Determine the Release

If this workflow was triggered by a release event:
- Use the release from `${{ github.event.release.tag_name }}`
- Get the release details using the `get_release` tool with the tag name

### Step 2: Gather Release Information

1. **Get the current release details** using `get_release`
   - Note the release name, tag, description, and publication date
   - Identify if this is a major, minor, or patch release based on semantic versioning

2. **Find the previous release** using `list_releases`
   - Get the list of releases and identify the previous release before the current one
   - This will help determine the range of changes to analyze

3. **Analyze the changes** between releases:
   - Use `list_commits` to get commits between the previous release and current release
   - Use `search_pull_requests` to find pull requests merged in this release period
   - Use `search_issues` to find issues closed in this release period

### Step 3: Analyze Code Changes

1. **Categorize the changes:**
   - **New Features**: Look for commits/PRs that add new functionality
   - **Bug Fixes**: Identify commits/PRs that resolve issues
   - **Performance Improvements**: Find optimizations and performance enhancements
   - **Documentation**: Note documentation updates and improvements
   - **Dependencies**: Track dependency updates and security fixes
   - **Breaking Changes**: Identify any breaking changes that users need to know about

2. **Identify key contributors** from commit authors and PR creators

3. **Find related issues and PRs** that provide context about the changes

### Step 4: Generate the Release Story

Create an engaging narrative that includes:

1. **Executive Summary** (2-3 sentences)
   - What this release achieves
   - Why users should be excited about it

2. **Key Highlights** (3-5 major items)
   - Most important new features
   - Critical bug fixes
   - Performance improvements
   - Breaking changes (if any)

3. **Detailed Walkthrough**
   - Group changes by category (Features, Fixes, Improvements, etc.)
   - For each major change, explain:
     - What it does
     - Why it matters
     - How users can benefit
     - Any migration steps needed

4. **Community Impact**
   - Acknowledge key contributors
   - Highlight community-driven improvements
   - Thank beta testers and issue reporters

5. **What's Next**
   - Hint at upcoming features
   - Roadmap direction
   - How users can contribute

### Step 5: Format the Story

Format the story using engaging Markdown with:
- **Emojis** to make sections visually appealing
- **Code examples** where relevant
- **Links** to related PRs, issues, and documentation
- **Clear sections** with descriptive headers
- **Bullet points** for easy scanning

### Step 6: Update the Release

Use the `update_release` tool to update the release description with your generated story. 

**Important Guidelines:**
- Keep the story informative but engaging
- Focus on user benefits rather than technical implementation details
- Use clear, jargon-free language
- Include actionable information (upgrade instructions, breaking changes)
- Make it scannable with good formatting
- Maintain a positive, professional tone
- Include relevant links for deeper exploration

Remember: This story will be the first thing users see when exploring this release. Make it count!

@include shared/include-link.md

@include shared/job-summary.md

@include shared/xpia.md

@include shared/gh-extra-tools.md