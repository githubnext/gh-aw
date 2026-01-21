const input = `\`\`\`markdown
You can create code blocks like this:
\`\`\`javascript
function hello() {
  console.log("world");
}
\`\`\`
\`\`\``;

console.log("Input:");
console.log(input);

const lines = input.split("\n");
const fences = [];
for (let i = 0; i < lines.length; i++) {
  const fenceMatch = lines[i].match(/^(\s*)(`{3,}|~{3,})([^`~\s]*)?(.*)$/);
  if (fenceMatch) {
    fences.push({
      lineIndex: i,
      indent: fenceMatch[1],
      char: fenceMatch[2][0],
      length: fenceMatch[2].length,
      language: fenceMatch[3] || "",
      trailing: fenceMatch[4] || "",
    });
  }
}

console.log("\nFences found:");
fences.forEach((f, idx) => {
  console.log(`  ${idx}: line ${f.lineIndex}, ${f.char.repeat(f.length)}${f.language}`);
});

console.log("\nProcessing fence 0 (```markdown):");
// Find potential closers for fence 0
const potentialClosers = [];
const openIndentLength = fences[0].indent.length;

for (let j = 1; j < fences.length; j++) {
  const fence = fences[j];
  const canClose = fence.char === fences[0].char && fence.length >= fences[0].length && fence.language === "";
  
  if (canClose && fence.indent.length === openIndentLength) {
    // Check if there's an opener between fence 0 and this closer
    let hasOpenerBetween = false;
    for (let k = 1; k < j; k++) {
      const intermediateFence = fences[k];
      if (intermediateFence.language !== "" && intermediateFence.indent.length === openIndentLength) {
        hasOpenerBetween = true;
        console.log("  Closer at fence " + j + " (line " + fence.lineIndex + ") has opener between: fence " + k + " (```" + intermediateFence.language + ")");
        break;
      }
    }
    
    if (!hasOpenerBetween) {
      console.log("  Closer at fence " + j + " (line " + fence.lineIndex + ") has NO opener between");
    }
    
    potentialClosers.push({ index: j, hasOpenerBetween });
  }
}

console.log("\nPotential closers:");
potentialClosers.forEach(c => {
  console.log("  Fence " + c.index + " (line " + fences[c.index].lineIndex + "), hasOpenerBetween: " + c.hasOpenerBetween);
});

const directClosers = potentialClosers.filter(c => !c.hasOpenerBetween);
console.log("\nDirect closers (no opener between): " + directClosers.length);
directClosers.forEach(c => {
  console.log("  Fence " + c.index + " (line " + fences[c.index].lineIndex + ")");
});

if (directClosers.length > 1) {
  console.log("\nMultiple direct closers → TRUE NESTING → ESCAPE");
} else if (directClosers.length === 1) {
  console.log("\nOne direct closer → NORMAL CASE → PAIR");
} else if (potentialClosers.length > 0 && potentialClosers[0].hasOpenerBetween) {
  console.log("\nFirst closer has opener between → SKIP, process intermediate opener first");
}
