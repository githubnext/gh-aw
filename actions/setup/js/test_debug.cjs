const balancer = require('./markdown_code_region_balancer.cjs');

const input = `\`\`\`javascript
js code
\`\`\`

\`\`\`python
py code
\`\`\`

\`\`\`typescript
ts code
\`\`\``;

console.log("Input:");
console.log(input);
console.log("\n\nOutput:");
const output = balancer.balanceCodeRegions(input);
console.log(output);
console.log("\n\nMatch:", input === output ? "✓" : "✗");
