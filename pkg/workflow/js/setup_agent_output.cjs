const fs = require("fs");

function main() {
  // Use fixed safe outputs filename
  const outputFile = "/tmp/safe-outputs/raw.jsonl";
  fs.mkdirSync("/tmp/safe-outputs", { recursive: true });
  core.exportVariable("GITHUB_AW_SAFE_OUTPUTS", outputFile);
  core.setOutput("output_file", outputFile);
}
main();
