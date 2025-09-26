const fs = require("fs");
function main() {
  const outputFile = `/tmp/safe-outputs/outputs.jsonl`;
  fs.mkdirSync("/tmp/safe-outputs", { recursive: true });
  core.exportVariable("GITHUB_AW_SAFE_OUTPUTS", outputFile);
  core.setOutput("output_file", outputFile);
}
main();
