const fs = require('fs');
const path = require('path');

try {
  // Get context data from GitHub Actions
  const workflowName = process.env.WORKFLOW_NAME || 'Unnamed Workflow';
  const workflowDescription = process.env.WORKFLOW_DESCRIPTION || 'No description provided';  
  const workflowMarkdown = process.env.WORKFLOW_MARKDOWN || 'No content provided';
  const agentOutput = process.env.AGENT_OUTPUT || '';
  const agentPatch = process.env.AGENT_PATCH || '';

  // Embedded template content (injected during compilation)
  const templateContent = `__TEMPLATE_CONTENT__`;

  // Create threat detection directories
  const promptsDir = '/tmp/threat-detection/prompts';
  fs.mkdirSync(promptsDir, { recursive: true });
  
  // Write template content with placeholder replacement
  let processedContent = templateContent
    .replace(/{WORKFLOW_NAME}/g, workflowName)
    .replace(/{WORKFLOW_DESCRIPTION}/g, workflowDescription)
    .replace(/{WORKFLOW_MARKDOWN}/g, workflowMarkdown)
    .replace(/{AGENT_OUTPUT}/g, agentOutput)
    .replace(/{AGENT_PATCH}/g, agentPatch);

  // Write processed template to file
  const promptFile = path.join(promptsDir, 'detection.md');
  fs.writeFileSync(promptFile, processedContent);

  // Set environment variable for subsequent steps
  if (typeof process.env.GITHUB_ENV !== 'undefined') {
    fs.appendFileSync(process.env.GITHUB_ENV, `GITHUB_AW_PROMPT=${promptFile}\n`);
  }

  console.log('Threat detection setup completed successfully');
  console.log(`Prompt file created at: ${promptFile}`);

} catch (error) {
  console.error('Failed to setup threat detection:', error.message);
  process.exit(1);
}