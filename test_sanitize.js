#!/usr/bin/env node

const fs = require('fs');
const path = require('path');

// Read the sanitize_output.cjs file
const sanitizeScript = fs.readFileSync(path.join(__dirname, 'pkg/workflow/js/sanitize_output.cjs'), 'utf8');

// Extract the sanitizeContent function by evaluating the script in a clean context
const vm = require('vm');
const context = vm.createContext({
  global: {},
  process: process,
  require: require,
  console: console
});

const scriptWithExport = sanitizeScript.replace('await main();', 'global.testSanitizeContent = sanitizeContent;');
vm.runInContext(scriptWithExport, context);

const sanitizeContent = context.global.testSanitizeContent;

// Test the specific pattern mentioned in the problem statement
const testCases = [
  '**word:**',
  '**example:**',
  '**test:** some content',
  'This is **word:** pattern',
  'Multiple **word1:** and **word2:** patterns',
  'Mixed: http://bad.com and **word:** and https://github.com/repo',
  'word:',
  '**word',
  'word:**',
  'normal text',
  'protocol:test', // This should be sanitized
  'javascript:alert()', // This should be sanitized
];

console.log('Testing sanitization behavior:');
console.log('================================');

testCases.forEach(testCase => {
  const result = sanitizeContent(testCase);
  const changed = result !== testCase;
  
  console.log(`Input:  "${testCase}"`);
  console.log(`Output: "${result}"`);
  console.log(`Changed: ${changed}`);
  console.log('---');
});