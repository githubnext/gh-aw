#!/usr/bin/env node
const fs = require("fs");
const path = require("path");

// Get all .js files in the current directory
const files = fs.readdirSync(__dirname).filter(f => f.endsWith(".js") && f !== "remove-empty-exports.js");

files.forEach(file => {
  const filePath = path.join(__dirname, file);
  let content = fs.readFileSync(filePath, "utf8");

  // Remove lines that are exactly "export {};" or "export {};\n"
  content = content.replace(/^export \{\};?\s*$/gm, "");

  fs.writeFileSync(filePath, content, "utf8");
});

console.log(`Processed ${files.length} files`);
