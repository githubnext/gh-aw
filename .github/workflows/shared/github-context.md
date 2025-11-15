---
# GitHub Context Share
# 
# Import this file to get comprehensive GitHub invocation context in your workflow.
# This provides a standardized way to access GitHub context information with 
# conditional rendering based on the event type.
#
# Usage:
#   imports:
#     - shared/github-context.md
#
# The context information will be automatically included in your workflow prompt
# with only relevant fields populated based on the triggering event.
---

## 📋 GitHub Invocation Context

### Repository Information
- **Repository**: ${{ github.repository }}
- **Owner**: ${{ github.owner }}
- **Server URL**: ${{ github.server_url }}
- **Workspace**: ${{ github.workspace }}

### Workflow Execution
- **Workflow Name**: ${{ github.workflow }}
- **Run ID**: [${{ github.run_id }}](${{ github.server_url }}/${{ github.repository }}/actions/runs/${{ github.run_id }})
- **Run Number**: #${{ github.run_number }}
- **Job ID**: ${{ github.job }}
- **Actor**: @${{ github.actor }}

### Event-Specific Context

The following context fields are populated based on the triggering event type. Fields not applicable to the current event will be empty.

#### Issue Context
- **Issue Number**: ${{ github.event.issue.number }}
- **Issue URL**: ${{ github.server_url }}/${{ github.repository }}/issues/${{ github.event.issue.number }}

#### Pull Request Context  
- **PR Number**: ${{ github.event.pull_request.number }}
- **PR URL**: ${{ github.server_url }}/${{ github.repository }}/pull/${{ github.event.pull_request.number }}

#### Comment Context
- **Comment ID**: ${{ github.event.comment.id }}

#### Review Context
- **Review ID**: ${{ github.event.review.id }}
- **Review Comment ID**: ${{ github.event.review_comment.id }}

#### Workflow Run Context
- **Workflow Run ID**: ${{ github.event.workflow_run.id }}

#### Release Context
- **Release ID**: ${{ github.event.release.id }}
- **Tag Name**: ${{ github.event.release.tag_name }}

#### Deployment Context
- **Deployment ID**: ${{ github.event.deployment.id }}
- **Deployment Status ID**: ${{ github.event.deployment_status.id }}

#### Check Context
- **Check Run ID**: ${{ github.event.check_run.id }}
- **Check Suite ID**: ${{ github.event.check_suite.id }}

#### Project Context
- **Label ID**: ${{ github.event.label.id }}
- **Milestone ID**: ${{ github.event.milestone.id }}
- **Project ID**: ${{ github.event.project.id }}
- **Project Card ID**: ${{ github.event.project_card.id }}
- **Project Column ID**: ${{ github.event.project_column.id }}

#### System Context
- **Organization ID**: ${{ github.event.organization.id }}
- **Repository ID**: ${{ github.event.repository.id }}
- **Sender ID**: ${{ github.event.sender.id }}
- **Installation ID**: ${{ github.event.installation.id }}
- **Page ID**: ${{ github.event.page.id }}

### Commit Context
- **Before Commit**: ${{ github.event.before }}
- **After Commit**: ${{ github.event.after }}
- **Head Commit ID**: ${{ github.event.head_commit.id }}

### Sanitized Content Access

For secure access to issue/PR/comment content, use:
```
${{ needs.activation.outputs.text }}
```

This provides automatically sanitized content with:
- @mention neutralization  
- Bot trigger protection
- XML tag safety
- URI filtering (only HTTPS from trusted domains)
- Content limits (0.5MB max, 65k lines max)
- Control character removal
