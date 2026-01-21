// Simplified version of the algorithm with logging
const input = `\`\`\`markdown
You can create code blocks like this:
\`\`\`javascript
function hello() {
  console.log("world");
}
\`\`\`
\`\`\``;

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
    });
  }
}

console.log("Fences:");
fences.forEach((f, idx) => {
  console.log("  " + idx + ": line " + f.lineIndex + ", " + f.char.repeat(f.length) + f.language);
});

const processed = new Set();
const pairedBlocks = [];

const isInsideBlock = (lineIndex) => {
  for (const block of pairedBlocks) {
    if (lineIndex > block.start && lineIndex < block.end) {
      return true;
    }
  }
  return false;
};

let i = 0;
let iteration = 0;
while (i < fences.length && iteration < 20) { // Safety limit
  iteration++;
  
  if (processed.has(i)) {
    console.log("\nIteration " + iteration + ": Skip fence " + i + " (already processed)");
    i++;
    continue;
  }

  const openFence = fences[i];
  console.log("\nIteration " + iteration + ": Process fence " + i + " (line " + openFence.lineIndex + ", ```" + openFence.language + ")");
  processed.add(i);

  // Find potential closers
  const potentialClosers = [];
  const openIndentLength = openFence.indent.length;

  for (let j = i + 1; j < fences.length; j++) {
    if (processed.has(j)) continue;

    const fence = fences[j];
    if (isInsideBlock(fence.lineIndex)) continue;

    const canClose = fence.char === openFence.char && fence.length >= openFence.length && fence.language === "";

    if (canClose && fence.indent.length === openIndentLength) {
      let hasOpenerBetween = false;
      for (let k = i + 1; k < j; k++) {
        if (processed.has(k)) continue;
        const intermediateFence = fences[k];
        if (intermediateFence.language !== "" && intermediateFence.indent.length === openIndentLength) {
          hasOpenerBetween = true;
          break;
        }
      }
      
      potentialClosers.push({ index: j, hasOpenerBetween });
    }
  }

  console.log("  Potential closers: " + potentialClosers.length);
  potentialClosers.forEach(c => {
    console.log("    Fence " + c.index + " (line " + fences[c.index].lineIndex + "), hasOpenerBetween=" + c.hasOpenerBetween);
  });

  if (potentialClosers.length > 0) {
    const firstCloser = potentialClosers[0];
    
    if (firstCloser.hasOpenerBetween) {
      console.log("  → SKIP (first closer has opener between)");
      i++;
    } else {
      const directClosers = potentialClosers.filter(c => !c.hasOpenerBetween);
      console.log("  Direct closers: " + directClosers.length);
      
      if (directClosers.length > 1) {
        console.log("  → ESCAPE (multiple direct closers)");
        const closerIndex = directClosers[directClosers.length - 1].index;
        processed.add(closerIndex);
        pairedBlocks.push({
          start: fences[i].lineIndex,
          end: fences[closerIndex].lineIndex,
        });
        console.log("  Paired " + i + " with " + closerIndex);
        i = closerIndex + 1;
      } else {
        console.log("  → PAIR (one direct closer)");
        const closerIndex = firstCloser.index;
        processed.add(closerIndex);
        pairedBlocks.push({
          start: fences[i].lineIndex,
          end: fences[closerIndex].lineIndex,
        });
        console.log("  Paired " + i + " with " + closerIndex);
        i = closerIndex + 1;
      }
    }
  } else {
    console.log("  → NO CLOSER");
    i++;
  }
  
  console.log("  Paired blocks: " + JSON.stringify(pairedBlocks));
}

console.log("\nFinal paired blocks:");
pairedBlocks.forEach(b => {
  console.log("  Lines " + b.start + "-" + b.end);
});
