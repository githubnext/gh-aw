#!/usr/bin/env bash

set -euo pipefail

skill_dir=$(CDPATH='' cd -- "$(dirname -- "$0")/.." && pwd)
repo_root=$(CDPATH='' cd -- "$skill_dir/../../.." && pwd)
work_dir=$(mktemp -d "$repo_root/.operational-value-designer-test.XXXXXX")
trap 'rm -rf "$work_dir"' EXIT HUP INT TERM

grep -Fq 'graders.<distinct-id>.script' "$skill_dir/SKILL.md"
grep -Fq '256 KiB (262,144 UTF-8 bytes, inclusive)' "$skill_dir/SKILL.md"
grep -Fq 'helpers.createSessionLookups(session)' "$skill_dir/SKILL.md"
grep -Fq 'Return a finite number, `null`, or an object with `value`' "$skill_dir/SKILL.md"
grep -Fq 'ordered array of metric objects' "$skill_dir/SKILL.md"
grep -Fq 'in an isolated downstream `graders` job' "$skill_dir/SKILL.md"
grep -Fq 'graders.operational-value.run' "$skill_dir/SKILL.md"
grep -Fq 'GH_AW_SESSION_FILE' "$skill_dir/SKILL.md"
grep -Fq 'GH_AW_GRADER_CONFIG' "$skill_dir/SKILL.md"
grep -Fq 'grader-shell-file-write' "$skill_dir/SKILL.md"
if grep -Fq 'Bash, `run`, file-backed scripts,' "$skill_dir/SKILL.md"; then
    printf 'obsolete grader prohibition remains in skill guidance\n' >&2
    exit 1
fi
grep -Fq 'ultimate operational condition' "$skill_dir/SKILL.md"
grep -Fq 'Select the furthest downstream rung that satisfies all three' "$skill_dir/SKILL.md"
grep -Fq 'observable at grading time, attributable to this run or its exact subject, and independently verifiable' "$skill_dir/SKILL.md"
grep -Fq 'why stronger downstream effects are unavailable' "$skill_dir/SKILL.md"
grep -Fq 'Never reward output merely for existing' "$skill_dir/SKILL.md"
grep -Fq 'Missing evidence for the selected rung returns `null`; it never causes runtime fallback to a weaker rung' "$skill_dir/SKILL.md"

for guidance in \
    AGENTS.md \
    .github/aw/instructions.md \
    .github/aw/intent.md \
    .github/aw/create-agentic-workflow.md \
    .github/aw/update-agentic-workflow.md \
    .github/skills/developer-internals/SKILL.md
do
    grep -Fq 'graders.<distinct-id>.script' "$repo_root/$guidance"
done

node - "$repo_root" <<'NODE'
const assert = require("node:assert/strict");
const fs = require("node:fs");
const path = require("node:path");
const root = process.argv[2];
const text = fs.readFileSync(path.join(root, ".github/skills/operational-value-designer/SKILL.md"), "utf8");
const scripts = [...text.matchAll(/^    script: \|\n((?:      [^\n]*\n|\n)+)/gm)];
assert.equal(scripts.length, 1, "the documented custom grader must remain executable");
const script = scripts[0][1].replace(/^      /gm, "");
const { runCustomGraders } = require(path.join(root, "actions/setup/js/custom_graders.cjs"));
const { executeOperationalValueEvaluator } = require(path.join(root, "actions/setup/js/operational_value_grader.cjs"));
const shell = [...text.matchAll(/```bash\n(#![\s\S]*?)```/g)][0][1];
const config = { repository: "OWNER/REPO", issueNumber: 123 };
const event = data => ({ type: "safe_output.result", data, provenance: { source: "safe_output", job: "safe_outputs" } });
const result = { type: "add_comment", repo: "OWNER/REPO", number: 123, url: "https://github.com/OWNER/REPO/issues/123#issuecomment-1" };
for (const [name, session, value, configuration = config] of [
  ["attained", [event(result)], 1],
  ["missed", [event({ ...result, number: 456 })], 0],
  ["unavailable", [], null],
  ["malformed", [event({ ...result, number: "123" })], null],
  ["empty-repository", [event({ ...result, repo: "" })], null],
  ["invalid-issue", [event({ ...result, number: 0 })], null],
  ["invalid-target", [event(result)], null, { ...config, repository: "" }],
]) {
  const graders = [{ id: "comment-target-match", source: "inline", direction: "higher_is_better", threshold: 1, config: configuration, script }];
  const first = runCustomGraders(session, graders).output.results[0];
  const second = runCustomGraders(session, graders).output.results[0];
  assert.equal(first.value, value, name);
  assert.equal(first.status, value === null ? "unavailable" : value === 1 ? "pass" : "fail", name);
  assert.deepEqual(first, second, `${name} must be deterministic`);
  const shellResult = executeOperationalValueEvaluator(shell, session, configuration);
  assert.deepEqual(shellResult, [{ id: "comment-target-match", value }], `${name}: documented Bash predicate`);
  assert.deepEqual(executeOperationalValueEvaluator(shell, session, configuration), shellResult, `${name}: deterministic Bash`);
}
NODE

# Keep workflow/evaluator pairing and session-file compatibility under regression.
evaluator_path=$("$skill_dir/scripts/operational-value-evaluator-path.sh" daily-file-diet)
[[ $evaluator_path == .github/graders/daily-file-diet-operational-value.sh ]]
if "$skill_dir/scripts/operational-value-evaluator-path.sh" ../escape >/dev/null 2>&1; then
    printf 'invalid workflow name was accepted\n' >&2
    exit 1
fi

valid_evaluator="$work_dir/valid.sh"
cat > "$valid_evaluator" <<'EOF'
#!/usr/bin/env bash

set -euo pipefail

[[ $# -eq 1 && "$1" == "$GH_AW_SESSION_FILE" && -f "$1" ]]
jq -s -e 'type == "array"' "$1" >/dev/null
printf 'verification diagnostic\n' >&2
case_name=$(printf '%s\n' "$GH_AW_GRADER_CONFIG" | jq -r '.case // "attained"')
case "$case_name" in
    attained)
        printf '%s\n' '[{"id":"triaged-issues","value":2.5},{"id":"net-cost","value":-3.25}]'
        ;;
    missed)
        printf '%s\n' '[{"id":"triaged-issues","value":0.5},{"id":"net-cost","value":-7.75}]'
        ;;
    unavailable|malformed)
        printf '%s\n' '[{"id":"triaged-issues","value":null},{"id":"net-cost","value":null}]'
        ;;
    *)
        exit 1
        ;;
esac
EOF
chmod +x "$valid_evaluator"

"$skill_dir/scripts/verify-operational-value-evaluator.sh" "$valid_evaluator" >/dev/null

fixtures="$work_dir/fixtures.json"
jq -n '
    def fixture($name; $expected): {
        name: $name, session: [], config: {case: $name}, expected: $expected
    };
    [
        fixture("attained"; [{id: "triaged-issues", value: 2.5}, {id: "net-cost", value: -3.25}]),
        fixture("missed"; [{id: "triaged-issues", value: 0.5}, {id: "net-cost", value: -7.75}]),
        fixture("unavailable"; [{id: "triaged-issues", value: null}, {id: "net-cost", value: null}]),
        fixture("malformed"; [{id: "triaged-issues", value: null}, {id: "net-cost", value: null}])
    ]
' > "$fixtures"
"$skill_dir/scripts/verify-operational-value-evaluator.sh" "$valid_evaluator" "$fixtures" >/dev/null

assert_fixtures_rejected() {
    local name=$1
    local invalid_fixtures=$2
    if "$skill_dir/scripts/verify-operational-value-evaluator.sh" "$valid_evaluator" "$invalid_fixtures" >/dev/null 2>&1; then
        printf 'invalid evaluator fixtures were accepted: %s\n' "$name" >&2
        exit 1
    fi
}

missing_fixture="$work_dir/missing-fixture.json"
jq 'map(select(.name != "malformed"))' "$fixtures" > "$missing_fixture"
assert_fixtures_rejected missing-required-case "$missing_fixture"

mismatched_fixture="$work_dir/mismatched-fixture.json"
jq 'map(if .name == "attained" then .expected[0].value = 0.5 else . end)' "$fixtures" > "$mismatched_fixture"
assert_fixtures_rejected mismatched-output "$mismatched_fixture"

reversed_fixture="$work_dir/reversed-fixture.json"
jq 'map(
    if .name == "attained" then .config.case = "missed" | .expected[0].value = 0
    elif .name == "missed" then .config.case = "attained" | .expected[0].value = 1
    else . end
)' "$fixtures" > "$reversed_fixture"
assert_fixtures_rejected reversed-semantics "$reversed_fixture"

contract_repo="$work_dir/contract-repo"
mkdir -p "$contract_repo/.github/graders" "$contract_repo/.github/workflows"
git -C "$contract_repo" init -q
git -C "$contract_repo" config user.email test@example.com
git -C "$contract_repo" config user.name "Operational Value Test"
printf '%s\n' '#!/usr/bin/env bash' > "$contract_repo/.github/graders/example-operational-value.sh"
printf '%s\n' '# Example' > "$contract_repo/.github/workflows/example.md"
git -C "$contract_repo" add .
git -C "$contract_repo" commit -qm baseline
baseline_commit=$(git -C "$contract_repo" rev-parse HEAD)

printf '%s\n' 'printf changed' >> "$contract_repo/.github/graders/example-operational-value.sh"
if git -C "$contract_repo" -c advice.detachedHead=false \
    --no-pager diff --quiet "$baseline_commit" -- .github/graders/example-operational-value.sh; then
    printf 'contract test evaluator did not change\n' >&2
    exit 1
fi
if (cd "$contract_repo" && "$skill_dir/scripts/verify-operational-value-contract-change.sh" "$baseline_commit") >/dev/null 2>&1; then
    printf 'unpaired evaluator change was accepted\n' >&2
    exit 1
fi
(cd "$contract_repo" && "$skill_dir/scripts/verify-operational-value-contract-change.sh" --correction "$baseline_commit") >/dev/null

printf '%s\n' 'Updated contract.' >> "$contract_repo/.github/workflows/example.md"
(cd "$contract_repo" && "$skill_dir/scripts/verify-operational-value-contract-change.sh" "$baseline_commit") >/dev/null

make_evaluator() {
    local name=$1
    local evaluator_output=$2
    local generated_evaluator="$work_dir/$name.sh"
    cat > "$generated_evaluator" <<EOF
#!/usr/bin/env bash
set -euo pipefail
cat <<'JSON'
$evaluator_output
JSON
EOF
    chmod +x "$generated_evaluator"
    printf '%s\n' "$generated_evaluator"
}

assert_rejected() {
    local name=$1
    local evaluator_output=$2
    local generated_evaluator
    generated_evaluator=$(make_evaluator "$name" "$evaluator_output")
    if "$skill_dir/scripts/verify-operational-value-evaluator.sh" "$generated_evaluator" >/dev/null 2>&1; then
        printf 'invalid evaluator output was accepted: %s\n' "$name" >&2
        exit 1
    fi
}

assert_rejected empty-array '[]'
assert_rejected top-level-object '{"id":"score","value":1}'
assert_rejected duplicate-ids '[{"id":"same","value":1},{"id":"same","value":0}]'
assert_rejected empty-id '[{"id":"   ","value":1}]'
assert_rejected boolean-value '[{"id":"score","value":true}]'
assert_rejected string-value '[{"id":"score","value":"1"}]'
assert_rejected extra-field '[{"id":"score","value":1,"message":"not allowed"}]'
assert_rejected multiple-documents $'[{"id":"first","value":1}]\n[{"id":"second","value":0}]'
assert_rejected invalid-json 'not json'

nul_evaluator="$work_dir/nul-output.sh"
cat > "$nul_evaluator" <<'EOF'
#!/usr/bin/env bash
printf '[{"id":"score","value":1}]\0'
EOF
chmod +x "$nul_evaluator"
if "$skill_dir/scripts/verify-operational-value-evaluator.sh" "$nul_evaluator" >/dev/null 2>&1; then
    printf 'evaluator output containing a NUL byte was accepted\n' >&2
    exit 1
fi

failed_evaluator="$work_dir/failed.sh"
cat > "$failed_evaluator" <<'EOF'
#!/usr/bin/env bash
exit 1
EOF
chmod +x "$failed_evaluator"
if "$skill_dir/scripts/verify-operational-value-evaluator.sh" "$failed_evaluator" >/dev/null 2>&1; then
    printf 'unsuccessful evaluator was accepted\n' >&2
    exit 1
fi

oversized_evaluator="$work_dir/oversized.sh"
cat > "$oversized_evaluator" <<'EOF'
#!/usr/bin/env bash
set -euo pipefail
printf '[{"id":"score","value":1}]'
dd if=/dev/zero bs=1048576 count=1 2>/dev/null | tr '\0' ' '
EOF
chmod +x "$oversized_evaluator"
if "$skill_dir/scripts/verify-operational-value-evaluator.sh" "$oversized_evaluator" >/dev/null 2>&1; then
    printf 'oversized evaluator output was accepted\n' >&2
    exit 1
fi

printf 'operational-value-designer skill tests passed\n'