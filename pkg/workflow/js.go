package workflow

import (
	_ "embed"
)

//go:embed js/create_pull_request.cjs
var createPullRequestScript string

//go:embed js/create_issue.cjs
var createIssueScript string

//go:embed js/create_comment.cjs
var createCommentScript string
