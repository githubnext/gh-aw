const { writeFileSecure } = require("./secure_write.cjs");

const CATEGORIES = [
  { key: "indexing", title: "Guard direct slice and string indexing", legacy: /\b(indexing|indexed (access|values)|bounds checks)\b/i },
  { key: "ignored-results", title: "Handle ignored errors and discarded results", legacy: /\b(ignored|unchecked|discarded|fmt output|error, result, and state hygiene)\b/i },
  { key: "function-length", title: "function-length refactoring", legacy: /\b(largefunc|long.function|function.length)\b/i },
];

function marker(key) {
  return `<!-- lint-monster-category: ${key} -->`;
}

function hasProvenance(issue) {
  const labels = (issue.labels || []).map(label => (typeof label === "string" ? label : label.name));
  return issue.user?.login === "github-actions[bot]" && ["automation", "lint", "cookie"].every(label => labels.includes(label)) && (issue.body || "").includes("<!-- gh-aw-workflow-id: lint-monster -->");
}

function persistentBody(body = "") {
  return body.replace(/^> - \[[ x]\] expires <!-- gh-aw-expires:[^\r\n]*? -->[^\r\n]*(?:\r?\n|$)/gm, "").replace(/<!--\s*gh-aw-expires:[\s\S]*?-->/g, "");
}

function planCategories(issues) {
  const openIssues = issues.filter(issue => !issue.pull_request && issue.state === "open" && hasProvenance(issue)).sort((a, b) => a.number - b.number);
  const scopes = new Map(
    openIssues.map(issue => {
      const explicit = [...new Set([...(issue.body || "").matchAll(/<!-- lint-monster-category: ([a-z0-9-]+) -->/g)].map(match => match[1]))];
      return [issue.number, explicit.length ? explicit : CATEGORIES.filter(category => category.legacy.test(issue.title)).map(category => category.key)];
    })
  );
  const categories = CATEGORIES.map(category => {
    const unambiguous = openIssues.filter(issue => scopes.get(issue.number).length === 1 && scopes.get(issue.number)[0] === category.key);
    const marked = unambiguous.filter(issue => (issue.body || "").includes(marker(category.key)));
    const legacy = unambiguous.filter(issue => !(issue.body || "").includes(marker(category.key)));
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
  const ambiguousIssues = openIssues.filter(issue => scopes.get(issue.number).length > 1).map(issue => ({ issue_number: issue.number, categories: scopes.get(issue.number) }));
  return { complete: true, categories, ambiguous_issues: ambiguousIssues, open_issues: openIssues };
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
    persistent_body: persistentBody(issue.body || ""),
    assignees: (issue.assignees || []).map(assignee => assignee.login),
  }));
  writeFileSecure(outputPath, `${JSON.stringify(plan, null, 2)}\n`);
  core.info(`Found ${plan.open_issues.length} open LintMonster issues; category inventory is complete.`);
  return plan;
}

module.exports = { CATEGORIES, marker, persistentBody, planCategories, collectIssues };
