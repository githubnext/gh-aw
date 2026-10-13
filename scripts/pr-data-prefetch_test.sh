#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
TEST_DIR="$(mktemp -d)"
trap 'rm -rf "$TEST_DIR"' EXIT
mkdir -p "$TEST_DIR/bin" "$TEST_DIR/agent"

# Check the Actions wiring as well as the shell body: a prefix restore must
# publish a new revision even when the restored entry exists but is invalid.
restore_step="$(sed -n '/      - name: Restore PR data from cache/,/      - name: Pre-fetch PR diff/p' "$ROOT/.github/workflows/pr-data-prefetch.yml")"
save_step="$(sed -n '/      - name: Save PR data to cache/,$p' "$ROOT/.github/workflows/pr-data-prefetch.yml")"
cache_key="pr-prefetch-v2-\${{ github.event.pull_request.number }}-\${{ github.event.pull_request.head.sha }}"
grep -Fq 'uses: actions/cache/restore@' <<< "$restore_step"
grep -Fxq "          key: $cache_key" <<< "$restore_step"
grep -Fxq "            $cache_key-" <<< "$restore_step"
grep -Fq 'uses: actions/cache/save@' <<< "$save_step"
grep -Fxq "          key: $cache_key-\${{ github.run_id }}-\${{ github.run_attempt }}" <<< "$save_step"
test "$(grep -c 'path: /tmp/gh-aw/agent' <<< "$restore_step")" = 1
test "$(grep -c 'path: /tmp/gh-aw/agent' <<< "$save_step")" = 1
if grep -q 'if:' <<< "$save_step"; then
  echo 'Revisioned cache saves must not be skipped on cache hits' >&2
  exit 1
fi
grep -Fq 'bash scripts/pr-data-prefetch_test.sh' "$ROOT/Makefile"
for workflow in pr-code-quality-reviewer ponytail-reviewer mattpocock-skills-reviewer impeccable-skills-reviewer; do
  grep -Fq "key: pr-prefetch-v2-\${{ github.event.pull_request.number" "$ROOT/.github/workflows/$workflow.md"
  grep -Fq "\${{ github.run_id }}-\${{ github.run_attempt }}" "$ROOT/.github/workflows/$workflow.md"
  grep -Fq 'restore-keys:' "$ROOT/.github/workflows/$workflow.md"
done

# Execute the workflow's real shell body with isolated files and a fake API.
awk '
  /^        run: \|$/ { body = 1; next }
  body && /^          / { sub(/^          /, ""); print; next }
  body { exit }
' "$ROOT/.github/workflows/pr-data-prefetch.yml" |
  sed "s|/tmp/gh-aw/agent|$TEST_DIR/agent|g" > "$TEST_DIR/prefetch.sh"
test -s "$TEST_DIR/prefetch.sh"
awk '
  /^    run: \|$/ { body = 1; next }
  body && /^      / { sub(/^      /, ""); print; next }
  body { exit }
' "$ROOT/.github/workflows/shared/pr-diff-data-fetch.md" |
  sed "s|/tmp/gh-aw/agent|$TEST_DIR/agent|g" > "$TEST_DIR/consumer.sh"
test -s "$TEST_DIR/consumer.sh"

cat > "$TEST_DIR/bin/gh" <<'MOCK'
#!/usr/bin/env bash
set -euo pipefail
printf '%s\n' "$*" >> "$TEST_DIR/calls"
case "$*" in
  *"/files?per_page=100"*)
    if [ "$FAIL_FILES" = 1 ]; then
      echo 'gh: API rate limit exceeded for installation (HTTP 403)' >&2
      exit 1
    fi
    [[ "$*" == *"--paginate --slurp"* ]]
    cat "$TEST_DIR/files.json"
    ;;
  *"/comments?per_page=100"*)
    if [ "$FAIL_COMMENTS" = 1 ]; then
      echo '{"id":99,"body":"partial response"}'
      echo 'gh: API rate limit exceeded for installation (HTTP 403)' >&2
      exit 1
    fi
    [[ "$*" == *"--paginate"* ]]
    # gh applies --jq to each page before concatenating the results.
    jq -c '.[][] | {id, path, line: (.line // .original_line), body: .body[:200], user: .user.login}' "$TEST_DIR/comments.json"
    ;;
  "pr view "*)
    if [[ "$*" == *"--json headRefOid --jq"* ]]; then
      jq -r '.pull_request.head.sha' "$GITHUB_EVENT_PATH"
    else
      [[ "$*" != *",files"* ]]
      jq '.pull_request | {number, title, body, headRefName: .head.ref,
        headRefOid: .head.sha, additions, deletions, changedFiles: .changed_files}' "$GITHUB_EVENT_PATH"
    fi
    ;;
  *)
    echo "Unexpected API request: $*" >&2
    exit 1
    ;;
esac
MOCK
chmod +x "$TEST_DIR/bin/gh"
export TEST_DIR
export PATH="$TEST_DIR/bin:$PATH"
export PR_NUMBER=42 PR_HEAD_SHA=head-42 GITHUB_REPOSITORY=owner/repo PR_DIFF_MAX_LINES=3000
export EXPR_GITHUB_REPOSITORY=owner/repo
export GITHUB_EVENT_PATH="$TEST_DIR/event.json" FAIL_FILES=0 FAIL_COMMENTS=0
# Do not expose ambient GitHub credentials to the fixture.
unset GH_TOKEN GITHUB_TOKEN

cat > "$GITHUB_EVENT_PATH" <<'JSON'
{"pull_request":{"number":42,"title":"Current title","body":"Current body","head":{"ref":"feature","sha":"head-42"},"additions":8,"deletions":3,"changed_files":5}}
JSON
cat > "$TEST_DIR/files.json" <<'JSON'
[
  [
    {"filename":"src/new.txt","status":"added","additions":2,"deletions":0,"patch":"@@ -0,0 +1,2 @@\n+one\n+two"},
    {"filename":"workflow.lock.yml","status":"modified","additions":4,"deletions":1,"patch":"@@ -1 +1 @@\n-generated\n+generated"},
    {"filename":"image.png","status":"modified","additions":0,"deletions":0}
  ],
  [
    {"filename":"src/renamed.txt","previous_filename":"src/old.txt","status":"renamed","additions":1,"deletions":1,"patch":"@@ -1 +1 @@\n-old\n+new"},
    {"filename":"src/deleted.txt","status":"removed","additions":0,"deletions":1,"patch":"@@ -1 +0,0 @@\n-deleted"}
  ]
]
JSON
jq -n '[[{id:1,path:"src/new.txt",line:2,body:("x" * 220),user:{login:"reviewer"}}],
        [{id:2,path:"src/renamed.txt",line:null,original_line:1,body:"Existing comment",user:{login:"other"}}]]' \
  > "$TEST_DIR/comments.json"

run_prefetch() {
  : > "$TEST_DIR/calls"
  bash "$TEST_DIR/prefetch.sh" > "$TEST_DIR/output" 2>&1
}
run_consumer() {
  : > "$TEST_DIR/calls"
  bash "$TEST_DIR/consumer.sh" > "$TEST_DIR/output" 2>&1
}

run_prefetch
test "$(wc -l < "$TEST_DIR/calls" | tr -d ' ')" = 2
jq -e '.number == 42 and .title == "Current title" and .headRefOid == "head-42" and
  .headRefName == "feature" and .changedFiles == 5 and (.files | length) == 5 and
  .files[3] == {path:"src/renamed.txt",additions:1,deletions:1}' "$TEST_DIR/agent/pr-meta.json" >/dev/null
jq -e 'length == 2 and (.[0].body | length) == 200 and .[1].line == 1 and .[1].user == "other"' \
  "$TEST_DIR/agent/pr-review-comments.json" >/dev/null
grep -q '^new file mode 100644$' "$TEST_DIR/agent/pr-diff.patch"
grep -q '^deleted file mode 100644$' "$TEST_DIR/agent/pr-diff.patch"
grep -q '^diff --git a/src/old.txt b/src/renamed.txt$' "$TEST_DIR/agent/pr-diff.patch"
if grep -q 'workflow.lock.yml\|image.png' "$TEST_DIR/agent/pr-diff.patch"; then
  echo 'Generated or binary files unexpectedly included in diff' >&2
  exit 1
fi
test "$(cat "$TEST_DIR/agent/pr-data-head-sha.txt")" = head-42
cp "$TEST_DIR/agent/pr-diff.patch" "$TEST_DIR/expected.patch"

# A warm diff survives files-endpoint quota exhaustion without re-reading it.
export FAIL_FILES=1
jq '.pull_request.title = "Updated title" | .pull_request.body = "Updated body"' "$GITHUB_EVENT_PATH" > "$TEST_DIR/event.tmp"
mv "$TEST_DIR/event.tmp" "$GITHUB_EVENT_PATH"
jq '.[1][0].body = "Fresh comment"' "$TEST_DIR/comments.json" > "$TEST_DIR/comments.tmp"
mv "$TEST_DIR/comments.tmp" "$TEST_DIR/comments.json"
run_prefetch
test "$(wc -l < "$TEST_DIR/calls" | tr -d ' ')" = 1
grep -q '/comments?per_page=100' "$TEST_DIR/calls"
cmp "$TEST_DIR/expected.patch" "$TEST_DIR/agent/pr-diff.patch"
jq -e '.title == "Updated title" and .body == "Updated body"' "$TEST_DIR/agent/pr-meta.json" >/dev/null
jq -e '.[1].body == "Fresh comment"' "$TEST_DIR/agent/pr-review-comments.json" >/dev/null
cp "$TEST_DIR/agent/pr-meta.json" "$TEST_DIR/expected-meta.json"

# A reviewer restoring an older revision refreshes mutable metadata itself.
jq '.title = "Stale title" | .body = "Stale body"' "$TEST_DIR/expected-meta.json" > "$TEST_DIR/agent/pr-meta.json"
run_consumer
test "$(wc -l < "$TEST_DIR/calls" | tr -d ' ')" = 1
jq -e '.number == 42 and .title == "Updated title" and .body == "Updated body" and (.files | length) == 5' \
  "$TEST_DIR/agent/pr-meta.json" >/dev/null
cmp "$TEST_DIR/expected.patch" "$TEST_DIR/agent/pr-diff.patch"
PR_HEAD_SHA="" run_consumer
test "$(wc -l < "$TEST_DIR/calls" | tr -d ' ')" = 3
jq -e '.title == "Updated title" and .body == "Updated body"' "$TEST_DIR/agent/pr-meta.json" >/dev/null

# Invalid event identities have actionable errors and never publish a marker.
cp "$GITHUB_EVENT_PATH" "$TEST_DIR/expected-event.json"
for field in number head.sha; do
  jq ".pull_request.$field = null" "$TEST_DIR/expected-event.json" > "$GITHUB_EVENT_PATH"
  if run_prefetch; then
    echo "Mismatched event unexpectedly succeeded: $field" >&2
    exit 1
  fi
  grep -q '::error::Event payload did not match PR #42 at head head-42' "$TEST_DIR/output"
  test ! -e "$TEST_DIR/agent/pr-data-head-sha.txt"
  echo head-42 > "$TEST_DIR/agent/pr-data-head-sha.txt"
  if run_consumer; then
    echo "Consumer accepted mismatched event: $field" >&2
    exit 1
  fi
  grep -q '::error::Failed to refresh metadata for PR #42 at head head-42' "$TEST_DIR/output"
done
cp "$TEST_DIR/expected-event.json" "$GITHUB_EVENT_PATH"

# Stale, wrong-PR, and incomplete cache entries must not bypass files failures.
for invalid_cache in stale wrong-pr wrong-meta-head malformed incomplete; do
  echo head-42 > "$TEST_DIR/agent/pr-data-head-sha.txt"
  cp "$TEST_DIR/expected-meta.json" "$TEST_DIR/agent/pr-meta.json"
  case "$invalid_cache" in
    stale) echo stale-head > "$TEST_DIR/agent/pr-data-head-sha.txt" ;;
    wrong-pr)
      echo head-42 > "$TEST_DIR/agent/pr-data-head-sha.txt"
      jq '.number = 99' "$TEST_DIR/agent/pr-meta.json" > "$TEST_DIR/meta.tmp"
      mv "$TEST_DIR/meta.tmp" "$TEST_DIR/agent/pr-meta.json"
      ;;
    wrong-meta-head)
      jq '.headRefOid = "other-head"' "$TEST_DIR/agent/pr-meta.json" > "$TEST_DIR/meta.tmp"
      mv "$TEST_DIR/meta.tmp" "$TEST_DIR/agent/pr-meta.json"
      ;;
    malformed) echo invalid-json > "$TEST_DIR/agent/pr-meta.json" ;;
    incomplete) rm "$TEST_DIR/agent/pr-diff.patch" ;;
  esac
  if run_prefetch; then
    echo "Invalid cache unexpectedly succeeded: $invalid_cache" >&2
    exit 1
  fi
  grep -q '::error::Failed to fetch files for PR #42: gh: API rate limit exceeded' "$TEST_DIR/output"
  test ! -e "$TEST_DIR/agent/pr-data-head-sha.txt"
  test "$(wc -l < "$TEST_DIR/calls" | tr -d ' ')" = 1
  echo head-42 > "$TEST_DIR/agent/pr-data-head-sha.txt"
  if [ "$invalid_cache" = stale ]; then
    echo stale-head > "$TEST_DIR/agent/pr-data-head-sha.txt"
  fi
  if run_consumer; then
    echo "Consumer accepted invalid cache: $invalid_cache" >&2
    exit 1
  fi
  grep -q '::error::Failed to fetch files for PR #42' "$TEST_DIR/output"
done

# A successful refetch repairs invalid metadata for the next prefix restore.
export FAIL_FILES=0
run_prefetch
jq -e '.number == 42 and .headRefOid == "head-42" and (.files | length) == 5' \
  "$TEST_DIR/agent/pr-meta.json" >/dev/null
export FAIL_FILES=1
run_prefetch
test "$(wc -l < "$TEST_DIR/calls" | tr -d ' ')" = 1

# Failed comment pagination must not publish a valid cache or an empty success.
export FAIL_FILES=0 FAIL_COMMENTS=1
if run_prefetch; then
  echo 'Failed review-comment fetch unexpectedly succeeded' >&2
  exit 1
fi
grep -q '::error::Failed to fetch review comments for PR #42' "$TEST_DIR/output"
test ! -e "$TEST_DIR/agent/pr-data-head-sha.txt"
jq -e '.[1].body == "Fresh comment"' "$TEST_DIR/agent/pr-review-comments.json" >/dev/null

# Empty diffs are valid (all generated/binary files); cap nonempty diffs exactly.
export FAIL_COMMENTS=0 PR_DIFF_MAX_LINES=4
run_prefetch
test "$(wc -l < "$TEST_DIR/agent/pr-diff.patch" | tr -d ' ')" = 4
rm "$TEST_DIR/agent/pr-data-head-sha.txt"
echo '[[{"filename":"image.png","status":"added","additions":0,"deletions":0}]]' > "$TEST_DIR/files.json"
run_prefetch
test ! -s "$TEST_DIR/agent/pr-diff.patch"
export FAIL_FILES=1
run_prefetch
test "$(wc -l < "$TEST_DIR/calls" | tr -d ' ')" = 1

echo 'PR data prefetch regression tests passed'
