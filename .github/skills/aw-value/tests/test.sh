#!/usr/bin/env bash

set -euo pipefail

skill_dir=$(CDPATH='' cd -- "$(dirname -- "$0")/.." && pwd)
repo_root=$(CDPATH='' cd -- "$skill_dir/../../.." && pwd)
work_dir=$(mktemp -d "$repo_root/.aw-value-test.XXXXXX")
trap 'rm -rf "$work_dir"' EXIT HUP INT TERM

path=$("$skill_dir/scripts/value-function-path.sh" daily-file-diet)
[[ $path == .github/graders/daily-file-diet-value.sh ]]
if "$skill_dir/scripts/value-function-path.sh" ../escape >/dev/null 2>&1; then
    printf 'invalid workflow name was accepted\n' >&2
    exit 1
fi

function_path="$work_dir/value.sh"
cat > "$function_path" <<'EOF'
#!/usr/bin/env bash

set -euo pipefail

case ${1:-} in
    --definition)
        cat <<'JSON'
{
  "schemaVersion": 4,
  "grader": "value",
  "repository": "owner/repo",
  "workflowName": "Example",
  "sourcePath": ".github/workflows/example.md",
  "adoption": {
    "commit": "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
    "adoptedAt": "2026-01-01T00:00:00Z"
  },
  "operationalValue": "For eligible issues, attain closure demonstrated by a closed issue.",
  "evidence": {
    "opportunity": "The issue assigned to the workflow run.",
    "assignment": "Bind the triggering issue number to the run ID.",
    "accepted": "The issue is closed by the evidence cutoff.",
    "repositories": ["owner/repo"],
    "collection": "Read immutable issue events through the capped cutoff.",
    "maturation": "Seven days after the run was created.",
    "zeroRule": "An eligible open issue at the cutoff scores zero.",
    "missingRule": "An unavailable issue or event history scores null."
  },
  "primaryMetric": {
    "id": "issue-closure",
    "formula": "1 when closed, otherwise 0",
    "direction": "higher_is_better"
  },
  "baseline": {
    "mode": "baseline-comparable",
    "value": 0.4,
    "evidenceCutoff": "2025-12-31T00:00:00Z",
    "provenance": [{"repository": "owner/repo", "kind": "issue-events", "ref": "baseline"}]
  },
  "validationExamples": {
    "targetAttained": {"eligible": true, "closed": true},
    "targetMissed": {"eligible": true, "closed": false},
    "missing": {"eligible": false},
    "malformed": {"eligible": "yes"}
  }
}
JSON
        ;;
    --metric)
        jq 'if (.eligible | type) != "boolean" or .eligible == false or (.closed | type) != "boolean" then null elif .closed then 1 else 0 end'
        ;;
    *)
        exit 1
        ;;
esac
EOF
chmod +x "$function_path"

"$skill_dir/scripts/verify-value-function.sh" "$function_path" >/dev/null
printf 'aw-value skill tests passed\n'