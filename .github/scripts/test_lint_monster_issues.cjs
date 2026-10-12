const test = require("node:test");
const assert = require("node:assert/strict");
const fs = require("node:fs");
const os = require("node:os");
const path = require("node:path");
const { CATEGORIES, marker, planCategories, collectIssues } = require("./lint_monster_issues.cjs");

const issue = (number, title, extra = {}) => ({ number, title: `[lint-monster] ${title}`, state: "open", body: "", ...extra });

test("all twelve source reports reuse the oldest tracker despite changing counts and titles", () => {
  const issues = [
    issue(65739, "[LintMonster] Guard direct slice and string indexing (1,755 findings)"),
    issue(65740, "[LintMonster] Handle fmt output errors (908 findings)"),
    issue(65996, "[LintMonster] Bound checks for indexing findings (1,755)"),
    issue(65997, "[LintMonster] Error, result, and state hygiene findings (1,149)"),
    issue(66759, "LintMonster: add bounds checks for indexed access (1,745 findings)"),
    issue(66760, "LintMonster: handle ignored results and related Go idioms (1,149 findings)"),
    issue(67086, "Add bounds checks for indexed values"),
    issue(67087, "Handle unchecked and discarded results"),
    issue(67325, "LintMonster: guard direct slice and string indexing findings"),
    issue(67326, "LintMonster: handle ignored errors and discarded results"),
    issue(67598, "LintMonster: Guard indexing and type assertions (1,841 findings)"),
    issue(67599, "LintMonster: Handle ignored errors and discarded results (786 findings)"),
  ];
  const plan = planCategories(issues.reverse());
  assert.equal(plan.categories[0].action, "update");
  assert.equal(plan.categories[0].issue_number, 65739);
  assert.deepEqual(plan.categories[0].candidate_issue_numbers, [65739, 65996, 66759, 67086, 67325, 67598]);
  assert.equal(plan.categories[1].action, "update");
  assert.equal(plan.categories[1].issue_number, 65740);
  assert.deepEqual(plan.categories[1].candidate_issue_numbers, [65740, 65997, 66760, 67087, 67326, 67599]);
  assert.equal(plan.open_issues.length, 12);
  assert.equal(Object.hasOwn(plan, "close_issue_numbers"), false);
});

test("a rerun with a marked open tracker plans only an update for that category", () => {
  for (const category of CATEGORIES) {
    const tracker = issue(42, category.title, { body: marker(category.key) });
    const first = planCategories([tracker]).categories.find(item => item.key === category.key);
    const rerun = planCategories([tracker]).categories.find(item => item.key === category.key);
    assert.deepEqual(rerun, first);
    assert.equal(rerun.action, "update");
    assert.equal(rerun.issue_number, 42);
    assert.deepEqual(rerun.candidate_issue_numbers, [42]);
  }
});

test("marked trackers take precedence over legacy titles without splitting path snapshots", () => {
  const plan = planCategories([
    issue(1, "long-function findings in pkg/cli"),
    issue(2, "largefunc findings in pkg/workflow"),
    issue(3, "function-length refactoring", { body: marker("function-length") }),
    issue(4, "renamed by a maintainer", { body: marker("indexing") }),
  ]);
  assert.equal(plan.categories[2].issue_number, 3);
  assert.deepEqual(plan.categories[2].candidate_issue_numbers, [3, 1, 2]);
  assert.equal(plan.categories[0].issue_number, 4);
});

test("an explicit category marker overrides misleading legacy title words", () => {
  const plan = planCategories([issue(1, "Handle ignored indexing errors", { body: marker("ignored-results") }), issue(2, "Guard indexing analyzer reliability", { body: marker("analyzer-reliability") })]);
  assert.equal(plan.categories[0].action, "create");
  assert.equal(plan.categories[1].issue_number, 1);
  assert.deepEqual(plan.categories[1].candidate_issue_numbers, [1]);
  assert.equal(plan.open_issues.length, 2);
});

test("closed issues, PRs, other workflows and ambiguous categories never become authoritative", () => {
  const plan = planCategories([issue(1, "Guard indexing", { state: "closed" }), issue(2, "Guard indexing", { pull_request: {} }), issue(3, "Guard indexing", { title: "[eslint-monster] Guard indexing" }), issue(4, "Review a new analyzer")]);
  assert.ok(plan.categories.every(category => category.action === "create" && category.issue_number === null));
  assert.deepEqual(
    plan.open_issues.map(item => item.number),
    [4]
  );
});

test("inventory uses all paginated open issues, including trackers past the first page", async t => {
  const dir = fs.mkdtempSync(path.join(os.tmpdir(), "lint-monster-issues-"));
  t.after(() => fs.rmSync(dir, { recursive: true, force: true }));
  const requests = [];
  const listForRepo = () => {};
  const outputPath = path.join(dir, "agent", "issues.json");
  const plan = await collectIssues({
    github: {
      rest: { issues: { listForRepo } },
      async paginate(method, request) {
        assert.equal(method, listForRepo);
        requests.push(request);
        return [...Array.from({ length: 100 }, (_, n) => issue(n + 1, "Other work", { title: "Unrelated" })), issue(101, "Guard indexing")];
      },
    },
    context: { repo: { owner: "github", repo: "gh-aw" } },
    core: { info() {} },
    outputPath,
  });
  assert.deepEqual(requests, [{ owner: "github", repo: "gh-aw", state: "open", per_page: 100 }]);
  assert.equal(plan.categories[0].action, "update");
  assert.equal(plan.categories[0].issue_number, 101);
  assert.equal(plan.complete, true);
  assert.deepEqual(JSON.parse(fs.readFileSync(outputPath, "utf8")), plan);
});

test("an API failure fails closed instead of writing an empty create plan", async t => {
  const dir = fs.mkdtempSync(path.join(os.tmpdir(), "lint-monster-issues-"));
  t.after(() => fs.rmSync(dir, { recursive: true, force: true }));
  const outputPath = path.join(dir, "issues.json");
  await assert.rejects(
    collectIssues({
      github: {
        rest: { issues: { listForRepo() {} } },
        async paginate() {
          throw new Error("API unavailable");
        },
      },
      context: { repo: { owner: "github", repo: "gh-aw" } },
      core: { info() {} },
      outputPath,
    }),
    /API unavailable/
  );
  assert.equal(fs.existsSync(outputPath), false);
});
