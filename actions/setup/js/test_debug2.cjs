const balancer = require('./markdown_code_region_balancer.cjs');

const input1 = `# Example

Here's how to use code blocks:

\`\`\`markdown
You can create code blocks like this:
\`\`\`javascript
function hello() {
  console.log("world");
}
\`\`\`
\`\`\`

Text after`;

console.log("Test 1: AI-generated code with nested markdown");
console.log("Input:");
console.log(input1);
console.log("\n\nOutput:");
const output1 = balancer.balanceCodeRegions(input1);
console.log(output1);
console.log("\n\nMatch:", input1 === output1 ? "✓" : "✗");

console.log("\n\n" + "=".repeat(60) + "\n\n");

const input2 = `\`\`\`markdown
# Tutorial

\`\`\`javascript
code here
\`\`\`

More text
\`\`\``;

console.log("Test 2: Deeply nested example");
console.log("Input:");
console.log(input2);
console.log("\n\nOutput:");
const output2 = balancer.balanceCodeRegions(input2);
console.log(output2);
console.log("\n\nMatch:", input2 === output2 ? "✓" : "✗");
