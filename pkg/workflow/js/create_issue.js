// Read the agent output content from environment variable
const outputContent = process.env.GITHUB_AW_AGENT_OUTPUT;
if (!outputContent) {
  console.log('No GITHUB_AW_AGENT_OUTPUT environment variable found');
  return;
}

if (outputContent.trim() === '') {
  console.log('Agent output content is empty');
  return;
}

console.log('Agent output content length:', outputContent.length);

// Parse the output to extract title and body
const lines = outputContent.split('\n');
let title = '';
let bodyLines = [];
let foundTitle = false;

for (let i = 0; i < lines.length; i++) {
  const line = lines[i].trim();
  
  // Skip empty lines until we find the title
  if (!foundTitle && line === '') {
    continue;
  }
  
  // First non-empty line becomes the title
  if (!foundTitle && line !== '') {
    // Remove markdown heading syntax if present
    title = line.replace(/^#+\s*/, '').trim();
    foundTitle = true;
    continue;
  }
  
  // Everything else goes into the body
  if (foundTitle) {
    bodyLines.push(lines[i]); // Keep original formatting
  }
}

// If no title was found, use a default
if (!title) {
  title = 'Agent Output';
}

// Apply title prefix if provided via environment variable
const titlePrefix = process.env.GITHUB_AW_ISSUE_TITLE_PREFIX;
if (titlePrefix && !title.startsWith(titlePrefix)) {
  title = titlePrefix + title;
}

// Prepare the body content
const body = bodyLines.join('\n').trim();

// Parse labels from environment variable (comma-separated string)
const labelsEnv = process.env.GITHUB_AW_ISSUE_LABELS;
const labels = labelsEnv ? labelsEnv.split(',').map(label => label.trim()).filter(label => label) : [];

console.log('Creating issue with title:', title);
console.log('Labels:', labels);
console.log('Body length:', body.length);

// Check if we're in an issue context (triggered by an issue event)
const parentIssueNumber = context.payload?.issue?.number;
let finalBody = body;

if (parentIssueNumber) {
  console.log('Detected issue context, parent issue #' + parentIssueNumber);
  
  // Add reference to parent issue in the child issue body
  if (finalBody.trim()) {
    finalBody = `Related to #${parentIssueNumber}\n\n${finalBody}`;
  } else {
    finalBody = `Related to #${parentIssueNumber}`;
  }
}

// Create the issue using GitHub API
const { data: issue } = await github.rest.issues.create({
  owner: context.repo.owner,
  repo: context.repo.repo,
  title: title,
  body: finalBody,
  labels: labels
});

console.log('Created issue #' + issue.number + ': ' + issue.html_url);

// If we have a parent issue, add a comment to it referencing the new child issue
if (parentIssueNumber) {
  try {
    await github.rest.issues.createComment({
      owner: context.repo.owner,
      repo: context.repo.repo,
      issue_number: parentIssueNumber,
      body: `Created related issue: #${issue.number}`
    });
    console.log('Added comment to parent issue #' + parentIssueNumber);
  } catch (error) {
    console.log('Warning: Could not add comment to parent issue:', error.message);
  }
}

// Set output for other jobs to use
core.setOutput('issue_number', issue.number);
core.setOutput('issue_url', issue.html_url);