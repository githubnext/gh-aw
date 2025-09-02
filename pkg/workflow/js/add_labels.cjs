async function main() {
  // Read the validated output content from environment variable
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

  // Parse the validated output JSON
  let validatedOutput;
  try {
    validatedOutput = JSON.parse(outputContent);
  } catch (error) {
    console.log('Error parsing agent output JSON:', error instanceof Error ? error.message : String(error));
    return;
  }

  if (!validatedOutput.items || !Array.isArray(validatedOutput.items)) {
    console.log('No valid items found in agent output');
    return;
  }

  // Find the add-issue-label item
  const labelsItem = validatedOutput.items.find(/** @param {any} item */ item => item.type === 'add-issue-label');
  if (!labelsItem) {
    console.log('No add-issue-label item found in agent output');
    return;
  }

  console.log('Found add-issue-label item:', { labelsCount: labelsItem.labels.length });

  // Read the allowed labels from environment variable (optional)
  const allowedLabelsEnv = process.env.GITHUB_AW_LABELS_ALLOWED;
  let allowedLabels = null;
  
  if (allowedLabelsEnv && allowedLabelsEnv.trim() !== '') {
    allowedLabels = allowedLabelsEnv.split(',').map(label => label.trim()).filter(label => label);
    if (allowedLabels.length === 0) {
      allowedLabels = null; // Treat empty list as no restrictions
    }
  }

  if (allowedLabels) {
    console.log('Allowed labels:', allowedLabels);
  } else {
    console.log('No label restrictions - any labels are allowed');
  }

  // Read the max limit from environment variable (default: 3)
  const maxCountEnv = process.env.GITHUB_AW_LABELS_MAX_COUNT;
  const maxCount = maxCountEnv ? parseInt(maxCountEnv, 10) : 3;
  if (isNaN(maxCount) || maxCount < 1) {
    core.setFailed(`Invalid max value: ${maxCountEnv}. Must be a positive integer`);
    return;
  }

  console.log('Max count:', maxCount);

  // Check if we're in an issue or pull request context
  const isIssueContext = context.eventName === 'issues' || context.eventName === 'issue_comment';
  const isPRContext = context.eventName === 'pull_request' || context.eventName === 'pull_request_review' || context.eventName === 'pull_request_review_comment';

  if (!isIssueContext && !isPRContext) {
    core.setFailed('Not running in issue or pull request context, skipping label addition');
    return;
  }

  // Determine the issue/PR number
  let issueNumber;
  let contextType;

  if (isIssueContext) {
    if (context.payload.issue) {
      issueNumber = context.payload.issue.number;
      contextType = 'issue';
    } else {
      core.setFailed('Issue context detected but no issue found in payload');
      return;
    }
  } else if (isPRContext) {
    if (context.payload.pull_request) {
      issueNumber = context.payload.pull_request.number;
      contextType = 'pull request';
    } else {
      core.setFailed('Pull request context detected but no pull request found in payload');
      return;
    }
  }

  if (!issueNumber) {
    core.setFailed('Could not determine issue or pull request number');
    return;
  }

  // Extract labels from the JSON item
  const requestedLabels = labelsItem.labels || [];
  console.log('Requested labels:', requestedLabels);

  // Check for label removal attempts (labels starting with '-')
  for (const label of requestedLabels) {
    if (label.startsWith('-')) {
      core.setFailed(`Label removal is not permitted. Found line starting with '-': ${label}`);
      return;
    }
  }

  // Validate that all requested labels are in the allowed list (if restrictions are set)
  let validLabels;
  if (allowedLabels) {
    validLabels = requestedLabels.filter(/** @param {string} label */ label => allowedLabels.includes(label));
  } else {
    // No restrictions, all requested labels are valid
    validLabels = requestedLabels;
  }

  // Remove duplicates from requested labels
  let uniqueLabels = [...new Set(validLabels)];

  // Enforce max limit
  if (uniqueLabels.length > maxCount) {
    console.log(`too many labels, keep ${maxCount}`)
    uniqueLabels = uniqueLabels.slice(0, maxCount);
  }

  if (uniqueLabels.length === 0) {
    console.log('No labels to add');
    core.setOutput('labels_added', '');
    await core.summary.addRaw(`
## Label Addition

No labels were added (no valid labels found in agent output).
`).write();
    return;
  }

  console.log(`Adding ${uniqueLabels.length} labels to ${contextType} #${issueNumber}:`, uniqueLabels);

  try {
    // Add labels using GitHub API
    await github.rest.issues.addLabels({
      owner: context.repo.owner,
      repo: context.repo.repo,
      issue_number: issueNumber,
      labels: uniqueLabels
    });

    console.log(`Successfully added ${uniqueLabels.length} labels to ${contextType} #${issueNumber}`);

    // Set output for other jobs to use
    core.setOutput('labels_added', uniqueLabels.join('\n'));

    // Write summary
    const labelsListMarkdown = uniqueLabels.map(label => `- \`${label}\``).join('\n');
    await core.summary.addRaw(`
## Label Addition

Successfully added ${uniqueLabels.length} label(s) to ${contextType} #${issueNumber}:

${labelsListMarkdown}
`).write();

  } catch (error) {
    const errorMessage = error instanceof Error ? error.message : String(error);
    console.error('Failed to add labels:', errorMessage);
    core.setFailed(`Failed to add labels: ${errorMessage}`);
  }
}
await main();