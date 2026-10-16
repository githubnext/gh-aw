#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
TEST_DIR="$(mktemp -d)"
trap 'rm -rf "$TEST_DIR"' EXIT
mkdir -p "$TEST_DIR/bin" "$TEST_DIR/agent" "$TEST_DIR/workspace"

awk '
  /^    run: \|$/ { body = 1; next }
  body && /^      / { sub(/^      /, ""); print; next }
  body && /^$/ { print; next }
  body { exit }
' "$ROOT/.github/workflows/design-decision-gate.md" |
  sed "s|/tmp/gh-aw/agent|$TEST_DIR/agent|g" > "$TEST_DIR/prefetch.sh"
test -s "$TEST_DIR/prefetch.sh"

cat > "$TEST_DIR/bin/gh" <<'MOCK'
#!/usr/bin/env bash
set -euo pipefail
printf '%s\n' "$*" >> "$TEST_DIR/calls"
case "$*" in
  "pr view "*)
    [[ "$*" == *",state" ]]
    if [ "$FAIL_PR" = 1 ]; then
      echo 'gh: API request failed (HTTP 403)' >&2
      exit 1
    fi
    jq -n --arg state "$PR_STATE" \
      '{number:42,title:"Implementation",body:"",labels:[],state:$state,
        baseRefName:"main",headRefName:"feature",author:null,url:""}'
    ;;
  "api --paginate repos/owner/repo/pulls/42/files?per_page=100 --jq .[]")
    printf '{"filename":"pkg/implementation.go","additions":553}\n'
    ;;
  "pr diff 42 --repo owner/repo")
    printf 'diff --git a/pkg/implementation.go b/pkg/implementation.go\n'
    ;;
  *)
    echo "Unexpected API request: $*" >&2
    exit 1
    ;;
esac
MOCK
chmod +x "$TEST_DIR/bin/gh"
export TEST_DIR PATH="$TEST_DIR/bin:$PATH"
export EXPR_GITHUB_EVENT_NAME=pull_request EXPR_GITHUB_REPOSITORY=owner/repo
export EXPR_GITHUB_WORKSPACE="$TEST_DIR/workspace" PR_NUMBER=42 PR_STATE=OPEN FAIL_PR=0
unset GH_TOKEN GITHUB_TOKEN

run_prefetch() {
  : > "$TEST_DIR/calls"
  rm -f "$TEST_DIR/agent/adr-prefetch-summary.json"
  bash "$TEST_DIR/prefetch.sh" > "$TEST_DIR/output" 2>&1
}

# An open PR still requires an ADR for the same 553-line implementation.
run_prefetch
test "$(wc -l < "$TEST_DIR/calls" | tr -d ' ')" = 3
jq -e '.pr_number == 42 and .requires_adr_by_default_volume and
  .default_business_additions == 553 and .diff_available and (has("skip_reason") | not)' \
  "$TEST_DIR/agent/adr-prefetch-summary.json" >/dev/null

# A stale open webhook must not cause ADR generation on a closed/deleted branch.
for state in CLOSED MERGED; do
  PR_STATE="$state" run_prefetch
  test "$(wc -l < "$TEST_DIR/calls" | tr -d ' ')" = 1
  jq -e --arg state "$state" '.pr_number == 42 and
    (.skip_reason | contains($state)) and .file_count == 0 and (.diff_available | not)' \
    "$TEST_DIR/agent/adr-prefetch-summary.json" >/dev/null
  jq -e 'length == 0' "$TEST_DIR/agent/pr-files.json" >/dev/null
  grep -q '::notice::Pull request #42' "$TEST_DIR/output"
done

# Preserve manual dispatch behavior, both with and without a PR number.
EXPR_GITHUB_EVENT_NAME=workflow_dispatch PR_NUMBER="" run_prefetch
test ! -s "$TEST_DIR/calls"
jq -e '.pr_number == null and .skip_reason == "workflow_dispatch did not include inputs.pr_number"' \
  "$TEST_DIR/agent/adr-prefetch-summary.json" >/dev/null
EXPR_GITHUB_EVENT_NAME=workflow_dispatch run_prefetch
jq -e '.requires_adr_by_default_volume and (has("skip_reason") | not)' \
  "$TEST_DIR/agent/adr-prefetch-summary.json" >/dev/null

# API failures cannot turn an unverified PR into a successful skip.
if FAIL_PR=1 run_prefetch; then
  echo 'PR metadata failure unexpectedly succeeded' >&2
  exit 1
fi
test ! -e "$TEST_DIR/agent/adr-prefetch-summary.json"
grep -q 'HTTP 403' "$TEST_DIR/output"
echo 'Design Decision Gate prefetch regressions passed'
