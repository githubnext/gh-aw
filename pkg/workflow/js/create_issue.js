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
let issue;

if (parentIssueNumber) {
  console.log('Detected issue context, parent issue #' + parentIssueNumber);
  
  try {
    // Get the parent issue's GraphQL node ID
    const parentIssueQuery = `
      query($owner: String!, $repo: String!, $number: Int!) {
        repository(owner: $owner, name: $repo) {
          issue(number: $number) {
            id
          }
        }
      }
    `;
    
    const parentIssueResult = await github.graphql(parentIssueQuery, {
      owner: context.repo.owner,
      repo: context.repo.repo,
      number: parentIssueNumber
    });
    
    const parentIssueId = parentIssueResult.repository.issue.id;
    console.log('Found parent issue GraphQL ID:', parentIssueId);
    
    // Get the repository's GraphQL node ID
    const repoQuery = `
      query($owner: String!, $repo: String!) {
        repository(owner: $owner, name: $repo) {
          id
        }
      }
    `;
    
    const repoResult = await github.graphql(repoQuery, {
      owner: context.repo.owner,
      repo: context.repo.repo
    });
    
    const repositoryId = repoResult.repository.id;
    console.log('Found repository GraphQL ID:', repositoryId);
    
    // Create the issue as a sub-issue using GraphQL mutation
    const createIssueMutation = `
      mutation($repositoryId: ID!, $title: String!, $body: String, $labelIds: [ID!], $parentIssueId: ID!) {
        createIssue(input: {
          repositoryId: $repositoryId,
          title: $title,
          body: $body,
          labelIds: $labelIds,
          parentIssueId: $parentIssueId
        }) {
          issue {
            id
            number
            url
          }
        }
      }
    `;
    
    // Get label IDs if labels are specified
    let labelIds = [];
    if (labels && labels.length > 0) {
      const labelsQuery = `
        query($owner: String!, $repo: String!) {
          repository(owner: $owner, name: $repo) {
            labels(first: 100) {
              nodes {
                id
                name
              }
            }
          }
        }
      `;
      
      const labelsResult = await github.graphql(labelsQuery, {
        owner: context.repo.owner,
        repo: context.repo.repo
      });
      
      const availableLabels = labelsResult.repository.labels.nodes;
      labelIds = labels
        .map(label => availableLabels.find(l => l.name.toLowerCase() === label.toLowerCase())?.id)
        .filter(id => id);
      
      console.log('Found label IDs:', labelIds);
    }
    
    const createIssueResult = await github.graphql(createIssueMutation, {
      repositoryId: repositoryId,
      title: title,
      body: body || '',
      labelIds: labelIds,
      parentIssueId: parentIssueId
    });
    
    issue = {
      number: createIssueResult.createIssue.issue.number,
      html_url: createIssueResult.createIssue.issue.url,
      id: createIssueResult.createIssue.issue.id
    };
    
    console.log('Created sub-issue #' + issue.number + ': ' + issue.html_url);
    console.log('Successfully linked to parent issue #' + parentIssueNumber);
    
  } catch (error) {
    console.log('Error creating sub-issue with GraphQL, falling back to regular issue creation:', error.message);
    
    // Fallback to regular issue creation with text reference
    let finalBody = body;
    if (finalBody.trim()) {
      finalBody = `Related to #${parentIssueNumber}\n\n${finalBody}`;
    } else {
      finalBody = `Related to #${parentIssueNumber}`;
    }
    
    const issueResult = await github.rest.issues.create({
      owner: context.repo.owner,
      repo: context.repo.repo,
      title: title,
      body: finalBody,
      labels: labels
    });
    
    issue = issueResult.data;
    console.log('Created regular issue #' + issue.number + ': ' + issue.html_url);
    
    // Add a comment to the parent issue
    try {
      await github.rest.issues.createComment({
        owner: context.repo.owner,
        repo: context.repo.repo,
        issue_number: parentIssueNumber,
        body: `Created related issue: #${issue.number}`
      });
      console.log('Added comment to parent issue #' + parentIssueNumber);
    } catch (commentError) {
      console.log('Warning: Could not add comment to parent issue:', commentError.message);
    }
  }
} else {
  // No parent issue context, create a regular issue
  const issueResult = await github.rest.issues.create({
    owner: context.repo.owner,
    repo: context.repo.repo,
    title: title,
    body: body,
    labels: labels
  });
  
  issue = issueResult.data;
  console.log('Created issue #' + issue.number + ': ' + issue.html_url);
}

// Set output for other jobs to use
core.setOutput('issue_number', issue.number);
core.setOutput('issue_url', issue.html_url);