#!/usr/bin/env node

const fs = require('fs');
const path = require('path');

// Load the grammar
const grammarPath = path.join(__dirname, 'grammars', 'agentic-workflow.tmLanguage.json');
const grammar = JSON.parse(fs.readFileSync(grammarPath, 'utf-8'));

console.log('✓ Grammar file loaded successfully');
console.log(`  Name: ${grammar.name}`);
console.log(`  Scope: ${grammar.scopeName}`);
console.log(`  File types: ${grammar.fileTypes.join(', ')}`);

// Test basic structure
if (!grammar.patterns) {
  console.error('✗ Missing patterns in grammar');
  process.exit(1);
}

if (!grammar.repository) {
  console.error('✗ Missing repository in grammar');
  process.exit(1);
}

// Check for required repository items
const requiredItems = ['frontmatter', 'markdown-content', 'include-directive', 'github-context-expression'];
for (const item of requiredItems) {
  if (!grammar.repository[item]) {
    console.error(`✗ Missing required repository item: ${item}`);
    process.exit(1);
  }
  console.log(`✓ Found repository item: ${item}`);
}

// Check for agentic-specific patterns
const yamlAgentic = grammar.repository['yaml-agentic-specific'];
if (!yamlAgentic || !yamlAgentic.patterns) {
  console.error('✗ Missing yaml-agentic-specific patterns');
  process.exit(1);
}

console.log(`✓ Found ${yamlAgentic.patterns.length} agentic-specific YAML patterns`);

// Check include directive pattern
const includePattern = grammar.repository['include-directive'].patterns[0];
if (!includePattern.match || !includePattern.match.includes('@include')) {
  console.error('✗ Include directive pattern incorrect');
  process.exit(1);
}

console.log('✓ Include directive pattern looks correct');

// Check GitHub context expression pattern
const contextPattern = grammar.repository['github-context-expression'].patterns[0];
if (!contextPattern.match || !contextPattern.match.includes('\\$\\{\\{')) {
  console.error('✗ GitHub context expression pattern incorrect');
  process.exit(1);
}

console.log('✓ GitHub context expression pattern looks correct');

console.log('\n🎉 All grammar validation tests passed!');