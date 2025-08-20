package workflow

import (
	_ "embed"
)

//go:embed js/create_pull_request.mjs
var createPullRequestScript string

//go:embed js/create_issue.mjs
var createIssueScript string

//go:embed js/create_comment.mjs
var createCommentScript string
