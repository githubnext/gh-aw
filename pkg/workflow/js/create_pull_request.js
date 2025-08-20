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
const titlePrefix = process.env.GITHUB_AW_PR_TITLE_PREFIX;
if (titlePrefix && !title.startsWith(titlePrefix)) {
  title = titlePrefix + title;
}

// Prepare the body content
const body = bodyLines.join('\n').trim();

// Parse labels from environment variable (comma-separated string)
const labelsEnv = process.env.GITHUB_AW_PR_LABELS;
const labels = labelsEnv ? labelsEnv.split(',').map(label => label.trim()).filter(label => label) : [];

console.log('Creating pull request with title:', title);
console.log('Labels:', labels);
console.log('Body length:', body.length);

// Generate unique branch name based on timestamp and title
const timestamp = Date.now();
const sanitizedTitle = title.toLowerCase()
  .replace(/[^a-z0-9\s-]/g, '') // Remove special characters
  .replace(/\s+/g, '-') // Replace spaces with hyphens
  .substring(0, 30); // Limit length
const branchName = `agent-pr-${timestamp}-${sanitizedTitle}`;

console.log('Generated branch name:', branchName);

// Get the current default branch to use as base
const { data: repo } = await github.rest.repos.get({
  owner: context.repo.owner,
  repo: context.repo.repo
});
const baseBranch = repo.default_branch;

console.log('Base branch:', baseBranch);

// Get the SHA of the base branch
const { data: baseRef } = await github.rest.git.getRef({
  owner: context.repo.owner,
  repo: context.repo.repo,
  ref: `heads/${baseBranch}`
});
const baseSha = baseRef.object.sha;

console.log('Base SHA:', baseSha);

// Create a new branch
try {
  await github.rest.git.createRef({
    owner: context.repo.owner,
    repo: context.repo.repo,
    ref: `refs/heads/${branchName}`,
    sha: baseSha
  });
  console.log('Created branch:', branchName);
} catch (error) {
  console.error('Failed to create branch:', error.message);
  throw error;
}

// Note: In a real implementation, we would apply the patch here
// For now, we'll create a minimal commit to demonstrate the PR creation
// The actual patch application would require downloading the artifact and applying it

// Check if patch file exists and apply it
const fs = require('fs');
let patchApplied = false;

try {
  if (fs.existsSync('/tmp/aw.patch')) {
    console.log('Patch file found, checking contents...');
    const patchContent = fs.readFileSync('/tmp/aw.patch', 'utf8');
    
    if (patchContent && patchContent.trim() && !patchContent.includes('Failed to generate patch')) {
      console.log('Valid patch content found, applying patch...');
      
      // Apply the patch using git apply
      const { execSync } = require('child_process');
      try {
        execSync('git apply /tmp/aw.patch', { stdio: 'inherit' });
        console.log('Patch applied successfully');
        patchApplied = true;
      } catch (applyError) {
        console.log('Failed to apply patch with git apply, trying git am...');
        try {
          execSync('git am /tmp/aw.patch', { stdio: 'inherit' });
          console.log('Patch applied successfully with git am');
          patchApplied = true;
        } catch (amError) {
          console.log('Failed to apply patch with git am, will create manual commit:', amError.message);
        }
      }
    } else {
      console.log('Patch file is empty or contains error message, skipping patch application');
    }
  } else {
    console.log('No patch file found at /tmp/aw.patch, will create manual commit');
  }
} catch (error) {
  console.log('Error checking for patch file:', error.message);
}

// If patch wasn't applied, create a simple file to demonstrate the branch has changes
if (!patchApplied) {
  try {
    // Create a simple file to demonstrate the branch has changes
    const fileContent = `# Agent Output\n\n${body}\n\nGenerated on: ${new Date().toISOString()}\n`;
    const encodedContent = Buffer.from(fileContent).toString('base64');
    
    // Create a file in the new branch
    await github.rest.repos.createOrUpdateFileContents({
      owner: context.repo.owner,
      repo: context.repo.repo,
      path: 'agent-output.md',
      message: `Add agent output: ${title}`,
      content: encodedContent,
      branch: branchName
    });
    
    console.log('Created file in branch');
  } catch (error) {
    console.error('Failed to create file in branch:', error.message);
    throw error;
  }
}

// Create the pull request
const { data: pullRequest } = await github.rest.pulls.create({
  owner: context.repo.owner,
  repo: context.repo.repo,
  title: title,
  body: body,
  head: branchName,
  base: baseBranch
});

console.log('Created pull request #' + pullRequest.number + ': ' + pullRequest.html_url);

// Add labels if specified
if (labels.length > 0) {
  try {
    await github.rest.issues.addLabels({
      owner: context.repo.owner,
      repo: context.repo.repo,
      issue_number: pullRequest.number,
      labels: labels
    });
    console.log('Added labels to pull request:', labels);
  } catch (error) {
    console.log('Warning: Could not add labels to pull request:', error.message);
  }
}

// Set output for other jobs to use
core.setOutput('pull_request_number', pullRequest.number);
core.setOutput('pull_request_url', pullRequest.html_url);
core.setOutput('branch_name', branchName);