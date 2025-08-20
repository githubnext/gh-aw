const fs = require('fs');

// Read the agent output from the environment variable
const outputFile = process.env.GITHUB_AW_OUTPUT;
if (!outputFile) {
  console.log('No GITHUB_AW_OUTPUT environment variable found');
  return;
}

// Check if the output file exists
if (!fs.existsSync(outputFile)) {
  console.log('Output file does not exist:', outputFile);
  return;
}

// Read the output content
const outputContent = fs.readFileSync(outputFile, 'utf8');
if (outputContent.trim() === '') {
  console.log('Output file is empty');
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

// Apply title prefix if provided
{{if .TitlePrefix}}
const titlePrefix = {{.TitlePrefix | toJSON}};
if (titlePrefix && !title.startsWith(titlePrefix)) {
  title = titlePrefix + title;
}
{{end}}

// Prepare the body content
const body = bodyLines.join('\n').trim();

// Prepare labels array
const labels = [
{{if .Labels}}
{{range .Labels}}  {{. | toJSON}},
{{end}}
].filter(label => label); // Remove any empty entries
{{else}}
];
{{end}}

console.log('Creating issue with title:', title);
console.log('Labels:', labels);
console.log('Body length:', body.length);

// Create the issue using GitHub API
const { data: issue } = await github.rest.issues.create({
  owner: context.repo.owner,
  repo: context.repo.repo,
  title: title,
  body: body,
  labels: labels
});

console.log('Created issue #' + issue.number + ': ' + issue.html_url);

// Set output for other jobs to use
core.setOutput('issue_number', issue.number);
core.setOutput('issue_url', issue.html_url);