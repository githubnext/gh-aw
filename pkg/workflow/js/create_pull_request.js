// Required Node.js modules
const fs = require('fs');
const crypto = require('crypto');
const { execSync } = require('child_process');

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

// Generate unique branch name using cryptographic random hex
const randomHex = crypto.randomBytes(8).toString('hex');
const workflowId = process.env.GITHUB_AW_WORKFLOW_ID || 'workflow';
const branchName = `${workflowId}/${randomHex}`;

console.log('Generated branch name:', branchName);

// Get the base branch from environment variable
const baseBranch = process.env.GITHUB_AW_BASE_BRANCH;
if (!baseBranch) {
  throw new Error('GITHUB_AW_BASE_BRANCH environment variable is required');
}

console.log('Base branch:', baseBranch);

// Create a new branch using git CLI
try {
  // Configure git (required for commits)
  execSync('git config --global user.email "action@github.com"', { stdio: 'inherit' });
  execSync('git config --global user.name "GitHub Action"', { stdio: 'inherit' });
  
  // Create and checkout new branch
  execSync(`git checkout -b ${branchName}`, { stdio: 'inherit' });
  console.log('Created and checked out branch:', branchName);
} catch (error) {
  console.error('Failed to create branch with git CLI:', error.message);
  throw error;
}

// Note: In a real implementation, we would apply the patch here
// For now, we'll create a minimal commit to demonstrate the PR creation
// The actual patch application would require downloading the artifact and applying it

// Check if patch file exists and apply it
let patchApplied = false;

try {
  if (fs.existsSync('/tmp/aw.patch')) {
    console.log('Patch file found, checking contents...');
    const patchContent = fs.readFileSync('/tmp/aw.patch', 'utf8');
    
    if (patchContent && patchContent.trim() && !patchContent.includes('Failed to generate patch')) {
      console.log('Valid patch content found, applying patch...');
      
      // Apply the patch using git apply
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
          console.log('Failed to apply patch with git am:', amError.message);
          throw new Error(`Failed to apply patch: ${amError.message}`);
        }
      }
    } else {
      console.log('Patch file is empty or contains error message');
      throw new Error('Patch file is empty or contains error message - cannot create pull request without changes');
    }
  } else {
    console.log('No patch file found at /tmp/aw.patch');
    throw new Error('No patch file found - cannot create pull request without changes');
  }
} catch (error) {
  console.error('Error handling patch file:', error.message);
  throw error;
}

// Commit the changes if patch was applied
if (patchApplied) {
  try {
    execSync('git add .', { stdio: 'inherit' });
    execSync(`git commit -m "Add agent output: ${title}"`, { stdio: 'inherit' });
    execSync(`git push origin ${branchName}`, { stdio: 'inherit' });
    console.log('Changes committed and pushed');
  } catch (error) {
    console.error('Failed to commit and push changes:', error.message);
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