const { runDailyGitHubDocsSEOOptimizerDriver } = require("../../actions/setup/js/daily_github_docs_seo_optimizer_driver_helpers.cjs");

runDailyGitHubDocsSEOOptimizerDriver()
  .then(result => {
    process.exit(result.exitCode);
  })
  .catch(error => {
    const message = error instanceof Error ? error.message : String(error);
    process.stderr.write(`[daily-github-docs-seo-optimizer-driver] ${message}\n`);
    process.exit(1);
  });
