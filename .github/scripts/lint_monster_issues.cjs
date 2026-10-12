const fs = require("node:fs");
const path = require("node:path");

const CATEGORIES = [
  { key: "indexing", title: "Guard direct slice and string indexing", legacy: /\b(indexing|indexed (access|values)|bounds checks)\b/i },
  { key: "ignored-results", title: "Handle ignored errors and discarded results", legacy: /\b(ignored|unchecked|discarded|fmt output|error, result, and state hygiene)\b/i },
  { key: "function-length", title: "function-length refactoring", legacy: /\b(largefunc|long.function|function.length)\b/i },
];

function marker(key) {
  return `<!-- lint-monster-category: ${key} -->`;
}

function planCategories(issues) {
  const openIssues = issues.filter(issue => !issue.pull_request && issue.state === "open" && issue.title.startsWith("[lint-monster] ")).sort((a, b) => a.number - b.number);
  const categories = CATEGORIES.map(category => {
    const marked = openIssues.filter(issue => (issue.body || "").includes(marker(category.key)));
    const legacy = openIssues.filter(issue => !/<!-- lint-monster-category: [a-z0-9-]+ -->/.test(issue.body || "") && category.legacy.test(issue.title));
    const candidates = [...new Map([...marked, ...legacy].map(issue => [issue.number, issue])).values()];
    const authoritative = candidates[0];
    return {
      key: category.key,
      title: category.title,
      marker: marker(category.key),
      action: authoritative ? "update" : "create",
      issue_number: authoritative?.number ?? null,
      candidate_issue_numbers: candidates.map(issue => issue.number),
    };
  });
  return { complete: true, categories, open_issues: openIssues };
}

async function collectIssues({ github, context, core, outputPath = "/tmp/gh-aw/agent/lint-issues.json" }) {
  // Do not use search: its indexing delay and result cap can hide an open tracker.
  const issues = await github.paginate(github.rest.issues.listForRepo, {
    ...context.repo,
    state: "open",
    per_page: 100,
  });
  const plan = planCategories(issues);
  plan.open_issues = plan.open_issues.map(issue => ({
    number: issue.number,
    title: issue.title,
    body: issue.body,
    assignees: (issue.assignees || []).map(assignee => assignee.login),
  }));
  fs.mkdirSync(path.dirname(outputPath), { recursive: true });
  fs.writeFileSync(outputPath, `${JSON.stringify(plan, null, 2)}\n`);
  core.info(`Found ${plan.open_issues.length} open LintMonster issues; category inventory is complete.`);
  return plan;
}

module.exports = { CATEGORIES, marker, planCategories, collectIssues };
