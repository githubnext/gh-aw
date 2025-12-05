package workflow

import (
"strings"
"testing"
)

// TestDeduplicateRequiresWithSingleAndDoubleQuotes tests that deduplicateRequires
// handles both single and double quoted require statements correctly
func TestDeduplicateRequiresWithSingleAndDoubleQuotes(t *testing.T) {
input := `const fs = require("fs");
const path = require('path');

function test() {
  const result = path.join("/tmp", "test");
  return fs.readFileSync(result);
}
`

output := deduplicateRequires(input)

t.Logf("Input:\n%s", input)
t.Logf("Output:\n%s", output)

// Check that both requires are present
if !strings.Contains(output, `const fs = require("fs");`) {
t.Error("fs require with double quotes should be present")
}

if !strings.Contains(output, `const path = require('path');`) &&
!strings.Contains(output, `const path = require("path");`) {
t.Error("path require should be present (with single or double quotes)")
}

// Check that path is defined before its use
fsIndex := strings.Index(output, "const fs")
pathIndex := strings.Index(output, "const path")
joinIndex := strings.Index(output, "path.join")

if pathIndex == -1 {
t.Error("path require is missing")
}
if joinIndex == -1 {
t.Error("path.join usage is missing")
}
if pathIndex > joinIndex {
t.Errorf("path require appears after path.join usage (path at %d, join at %d)", pathIndex, joinIndex)
}
if fsIndex == -1 {
t.Error("fs require is missing")
}
}
