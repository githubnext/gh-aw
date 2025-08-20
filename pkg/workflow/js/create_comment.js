// Read the agent output content from environment variable
const outputContent = process.env.AGENT_OUTPUT_CONTENT;
if (!outputContent) {
  console.log('No AGENT_OUTPUT_CONTENT environment variable found');
  return;
}

if (outputContent.trim() === '') {
  console.log('Agent output content is empty');
  return;
}

console.log('Agent output content length:', outputContent.length);

// Check if we're in an issue or pull request context
const isIssueContext = context.eventName === 'issues' || context.eventName === 'issue_comment';
const isPRContext = context.eventName === 'pull_request' || context.eventName === 'pull_request_review' || context.eventName === 'pull_request_review_comment';

if (!isIssueContext && !isPRContext) {
  console.log('Not running in issue or pull request context, skipping comment creation');
  return;
}

// Determine the issue/PR number and comment endpoint
let issueNumber;
let commentEndpoint;

if (isIssueContext) {
  if (context.payload.issue) {
    issueNumber = context.payload.issue.number;
    commentEndpoint = 'issues';
  } else {
    console.log('Issue context detected but no issue found in payload');
    return;
  }
} else if (isPRContext) {
  if (context.payload.pull_request) {
    issueNumber = context.payload.pull_request.number;
    commentEndpoint = 'issues'; // PR comments use the issues API endpoint
  } else {
    console.log('Pull request context detected but no pull request found in payload');
    return;
  }
}

if (!issueNumber) {
  console.log('Could not determine issue or pull request number');
  return;
}

console.log(`Creating comment on ${commentEndpoint} #${issueNumber}`);
console.log('Comment content length:', outputContent.length);

// Create the comment using GitHub API
const { data: comment } = await github.rest.issues.createComment({
  owner: context.repo.owner,
  repo: context.repo.repo,
  issue_number: issueNumber,
  body: outputContent
});

console.log('Created comment #' + comment.id + ': ' + comment.html_url);

// Set output for other jobs to use
core.setOutput('comment_id', comment.id);
core.setOutput('comment_url', comment.html_url);