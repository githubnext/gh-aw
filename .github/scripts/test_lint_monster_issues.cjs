const test = require("node:test");
const assert = require("node:assert/strict");
const fs = require("node:fs");
const os = require("node:os");
const path = require("node:path");
const { CATEGORIES, marker, persistentBody, planCategories, collectIssues } = require("./lint_monster_issues.cjs");

const footer = "<!-- gh-aw-workflow-id: lint-monster -->";
const issue = (number, title, extra = {}) => ({
  number,
  title,
  state: "open",
  user: { login: "github-actions[bot]" },
  labels: [{ name: "automation" }, { name: "lint" }, { name: "cookie" }],
  ...extra,
  body: `${extra.body || ""}\n${footer}`,
});

test("all twelve source reports reuse the oldest tracker despite changing counts and titles", () => {
  const issues = [
    issue(65739, "[lint-monster] [LintMonster] Guard direct slice and string indexing (1,755 findings)"),
    issue(65740, "[lint-monster] [LintMonster] Handle fmt output errors (908 findings)"),
    issue(65996, "[lint-monster] [LintMonster] Bound checks for indexing findings (1,755)"),
    issue(65997, "[lint-monster] [LintMonster] Error, result, and state hygiene findings (1,149)"),
    issue(66759, "[lint-monster] LintMonster: add bounds checks for indexed access (1,745 findings)"),
    issue(66760, "[lint-monster] LintMonster: handle ignored results and related Go idioms (1,149 findings)"),
    issue(67086, "[lint-monster] Add bounds checks for indexed values"),
    issue(67087, "[lint-monster] Handle unchecked and discarded results"),
    issue(67325, "[lint-monster] LintMonster: guard direct slice and string indexing findings"),
    issue(67326, "[lint-monster] LintMonster: handle ignored errors and discarded results"),
    issue(67598, "[lint-monster] LintMonster: Guard indexing and type assertions (1,841 findings)"),
    issue(67599, "[lint-monster] LintMonster: Handle ignored errors and discarded results (786 findings)"),
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
  const withoutPrefix = issues.map(item => ({ ...item, title: item.title.replace("[lint-monster] ", "") }));
  assert.deepEqual(planCategories(withoutPrefix).categories, plan.categories);
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

test("closed issues, PRs, other workflows and unmatched titles never become authoritative", () => {
  const otherWorkflow = issue(3, "[eslint-monster] Guard indexing");
  otherWorkflow.body = "<!-- gh-aw-workflow-id: eslint-monster -->";
  const plan = planCategories([issue(1, "Guard indexing", { state: "closed" }), issue(2, "Guard indexing", { pull_request: {} }), otherWorkflow, issue(4, "Review a new analyzer")]);
  assert.ok(plan.categories.every(category => category.action === "create" && category.issue_number === null));
  assert.deepEqual(
    plan.open_issues.map(item => item.number),
    [4]
  );
});

test("spoofed titles and markers cannot outrank a tracker without workflow provenance", () => {
  const genuine = issue(42, "Guard indexing");
  const spoof = issue(1, "[lint-monster] Guard indexing", { body: marker("indexing") });
  for (const untrusted of [
    { ...spoof, user: { login: "public-author" } },
    { ...spoof, user: undefined },
    { ...spoof, labels: [{ name: "lint" }] },
    { ...spoof, labels: undefined },
    { ...spoof, body: marker("indexing") },
    { ...spoof, body: `${marker("indexing")}\n<!-- gh-aw-workflow-id: other-monster -->` },
  ]) {
    const plan = planCategories([untrusted, genuine]);
    assert.equal(plan.categories[0].issue_number, 42);
    assert.deepEqual(plan.categories[0].candidate_issue_numbers, [42]);
    assert.deepEqual(
      plan.open_issues.map(item => item.number),
      [42]
    );
  }
});

test("mixed legacy titles and multiple markers are inspection-only, never authoritative", () => {
  const plan = planCategories([
    issue(1, "Handle ignored indexing errors"),
    issue(2, "[lint-monster] Guard indexing and ignored error results"),
    issue(3, "Mixed work", { body: `${marker("indexing")}\n${marker("ignored-results")}` }),
    issue(4, "Guard indexing"),
    issue(5, "Handle ignored errors"),
  ]);
  assert.deepEqual(
    plan.ambiguous_issues,
    [1, 2, 3].map(number => ({ issue_number: number, categories: ["indexing", "ignored-results"] }))
  );
  assert.equal(plan.categories[0].issue_number, 4);
  assert.equal(plan.categories[1].issue_number, 5);
  assert.deepEqual(plan.categories[0].candidate_issue_numbers, [4]);
  assert.deepEqual(plan.categories[1].candidate_issue_numbers, [5]);
  const ambiguousOnly = planCategories(plan.open_issues.slice(0, 3));
  assert.ok(ambiguousOnly.categories.every(category => category.action === "create"));
});

test("a closed completed tracker does not prevent planning a new tracker for recurrence", () => {
  const plan = planCategories([issue(1, "Guard direct slice and string indexing", { state: "closed", body: marker("indexing") })]);
  assert.equal(plan.categories[0].action, "create");
  assert.equal(plan.categories[0].issue_number, null);
});

test("persistent bodies remove expiration metadata without losing maintainer content", () => {
  const notes = "Maintainer notes\n- [x] Completed work\n- [ ] Outstanding work\n";
  const body = `${notes}> - [x] expires <!-- gh-aw-expires: 2026-10-12T03:00:41.459Z --> on Oct 11, 2026\n${footer}\nKeep this <!-- gh-aw-expires: 2026-10-12 --> note`;
  assert.equal(persistentBody(body), `${notes}${footer}\nKeep this  note`);
  assert.equal(persistentBody(notes), notes);
  assert.equal(persistentBody(), "");
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
        return [...Array.from({ length: 100 }, (_, n) => ({ number: n + 1, title: "Unrelated", state: "open" })), issue(101, "Guard indexing", { body: "<!-- gh-aw-expires: 2026-10-12 -->" })];
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
  assert.ok(plan.open_issues[0].body.includes("gh-aw-expires"));
  assert.equal(plan.open_issues[0].persistent_body.includes("gh-aw-expires"), false);
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

async function collectEmptyIssues(outputPath) {
  return collectIssues({
    github: {
      rest: { issues: { listForRepo() {} } },
      async paginate() {
        return [];
      },
    },
    context: { repo: { owner: "github", repo: "gh-aw" } },
    core: { info() {} },
    outputPath,
  });
}

test("inventory replaces a destination symlink without overwriting its target", async t => {
  const dir = fs.mkdtempSync(path.join(os.tmpdir(), "lint-monster-issues-"));
  t.after(() => fs.rmSync(dir, { recursive: true, force: true }));
  const outside = path.join(dir, "outside.json");
  fs.writeFileSync(outside, "do-not-touch");
  const outputPath = path.join(dir, "issues.json");
  fs.symlinkSync(outside, outputPath);

  const plan = await collectEmptyIssues(outputPath);

  assert.equal(fs.readFileSync(outside, "utf8"), "do-not-touch");
  const fd = fs.openSync(outputPath, fs.constants.O_RDONLY | fs.constants.O_NOFOLLOW);
  try {
    const stat = fs.fstatSync(fd);
    assert.equal(stat.isFile(), true);
    assert.equal(stat.mode & 0o777, 0o600);
    const content = fs.readFileSync(fd, "utf8");
    assert.deepEqual(JSON.parse(content), plan);
    assert.ok(content.endsWith("\n"));
  } finally {
    fs.closeSync(fd);
  }
});

test("inventory rejects symlinked ancestors without creating files through them", async t => {
  const dir = fs.mkdtempSync(path.join(os.tmpdir(), "lint-monster-issues-"));
  t.after(() => fs.rmSync(dir, { recursive: true, force: true }));
  const outside = path.join(dir, "outside");
  fs.mkdirSync(outside);
  const linked = path.join(dir, "linked");
  fs.symlinkSync(outside, linked, "dir");
  const outputPath = path.join(linked, "agent", "issues.json");

  await assert.rejects(collectEmptyIssues(outputPath), /symlinked component/);

  assert.deepEqual(fs.readdirSync(outside), []);
});
