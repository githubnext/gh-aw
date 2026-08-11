#!/bin/bash
set +o histexpand
set -euo pipefail

WITH_TESTS=0
JS_TEST_EXCLUDES=(
    --exclude '**/*.integration.test.cjs'
    --exclude '**/frontmatter_hash_github_api.test.cjs'
)
if [ "${1:-}" = "--with-tests" ]; then
    WITH_TESTS=1
elif [ "$#" -ne 0 ]; then
    echo "Usage: scripts/agent-report-progress.sh [--with-tests]" >&2
    exit 1
fi

PARALLEL_DIR=$(mktemp -d)
job_pids=()
job_labels=()
job_logs=()

cleanup() {
    rm -rf "$PARALLEL_DIR"
}
trap cleanup EXIT

start_job() {
    local label="$1"
    shift
    local index="${#job_pids[@]}"
    local log_file="$PARALLEL_DIR/job-$index.log"

    echo "Starting $label..."
    ("$@") >"$log_file" 2>&1 &
    job_pids+=("$!")
    job_labels+=("$label")
    job_logs+=("$log_file")
}

wait_for_jobs() {
    local failed=0
    local index

    for index in "${!job_pids[@]}"; do
        if wait "${job_pids[$index]}"; then
            echo "✓ ${job_labels[$index]}"
        else
            echo "✗ ${job_labels[$index]}" >&2
            failed=1
        fi
        cat "${job_logs[$index]}"
    done

    job_pids=()
    job_labels=()
    job_logs=()
    return "$failed"
}

BASE_REF="${BASE_REF:-origin/main}"
BASE_COMMIT=$(git merge-base "$BASE_REF" HEAD 2>/dev/null || true)
if [ -z "$BASE_COMMIT" ]; then
    echo "Error: unable to determine merge-base from BASE_REF=$BASE_REF." >&2
    echo "Set BASE_REF explicitly, for example: BASE_REF=origin/main make agent-report-progress" >&2
    exit 1
fi

mapfile -t CHANGED_FILES < <(
    {
        git diff --name-only --diff-filter=ACDMR "$BASE_COMMIT"
        git ls-files --others --exclude-standard
    } | sed '/^$/d' | LC_ALL=C sort -u
)

if [ "${#CHANGED_FILES[@]}" -eq 0 ]; then
    echo "No changes relative to $BASE_REF; skipping pre-PR validation."
    exit 0
fi

go_files=()
go_packages=()
prettier_files=()
setup_js_files=()
eslint_factory_files=()
action_shell_files=()
workflow_drift_required=0
model_alias_validation_required=0

for file in "${CHANGED_FILES[@]}"; do
    case "$file" in
        *.go)
            package_dir=$(dirname "$file")
            # Analyzer fixtures under testdata/ are excluded from ./... package
            # patterns, so they must not be linted or tested as real packages.
            case "$file" in
                testdata/*|*/testdata/*) package_dir="" ;;
            esac
            if [ -n "$package_dir" ] && find "$package_dir" -maxdepth 1 -type f -name '*.go' -print -quit 2>/dev/null | grep -q .; then
                go_packages+=("./$package_dir")
            fi
            if [ -f "$file" ]; then
                go_files+=("$file")
            fi
            ;;
    esac

    case "$file" in
        *.cjs|*.js|*.mjs|*.ts|*.json)
            if [ -f "$file" ]; then
                prettier_files+=("$file")
            fi
            ;;
    esac

    case "$file" in
        actions/setup/js/*.cjs|actions/setup/js/*.js|actions/setup/js/*.mjs|actions/setup/js/*.ts)
            [ -f "$file" ] && setup_js_files+=("${file#actions/setup/js/}")
            ;;
        eslint-factory/*.cjs|eslint-factory/*.js|eslint-factory/*.mjs|eslint-factory/*.ts)
            [ -f "$file" ] && eslint_factory_files+=("$file")
            ;;
        actions/setup/sh/*.sh)
            [ -f "$file" ] && action_shell_files+=("$file")
            ;;
    esac

    case "$file" in
        .github/workflows/*.md|.github/workflows/*.lock.yml|cmd/*.go|pkg/*.go|go.mod|go.sum)
            workflow_drift_required=1
            ;;
    esac

    case "$file" in
        pkg/cli/data/models.json|actions/setup/js/models.json)
            model_alias_validation_required=1
            ;;
    esac
done

echo "Validating ${#CHANGED_FILES[@]} changed file(s) relative to $BASE_REF..."

CHECK_STALE_LOCK_BASE_REF="$BASE_REF" make --no-print-directory check-stale-lock-files

if [ "${#go_files[@]}" -gt 0 ]; then
    echo "Formatting ${#go_files[@]} changed Go file(s)..."
    gofmt -w "${go_files[@]}"
fi

if [ "${#prettier_files[@]}" -gt 0 ]; then
    echo "Formatting ${#prettier_files[@]} changed JavaScript/TypeScript/JSON file(s)..."
    npx prettier --write "${prettier_files[@]}" --ignore-path .prettierignore --log-level=error
fi

git diff --check "$BASE_COMMIT"

echo "Building gh-aw..."
make --no-print-directory build

mapfile -t go_packages < <(printf '%s\n' "${go_packages[@]}" | sed '/^$/d' | LC_ALL=C sort -u)

lint_go_packages() {
    GOPATH=$(go env GOPATH)
    if command -v golangci-lint >/dev/null 2>&1 || [ -x "$GOPATH/bin/golangci-lint" ]; then
        PATH="$GOPATH/bin:$PATH" golangci-lint run "${go_packages[@]}"
    else
        echo "golangci-lint is not installed. Run 'make deps-dev' to install dependencies." >&2
        return 1
    fi
}

lint_javascript() {
    (cd eslint-factory && npm run build --silent)

    if [ "${#setup_js_files[@]}" -gt 0 ]; then
        (
            cd actions/setup/js
            ../../../eslint-factory/node_modules/.bin/eslint \
                --config ../../../eslint-factory/eslint.config.cjs \
                "${setup_js_files[@]}"
        )
    fi
}

lint_action_shell_files() {
    make --no-print-directory lint-action-sh
    GOPATH=$(go env GOPATH)
    if command -v shellcheck >/dev/null 2>&1 || [ -x "$GOPATH/bin/shellcheck" ]; then
        PATH="$GOPATH/bin:$PATH" shellcheck --severity=error "${action_shell_files[@]}"
    else
        echo "shellcheck is not installed. Run 'make deps-dev' to install dependencies." >&2
        return 1
    fi
}

test_setup_javascript() {
    (
        cd actions/setup/js
        npm run typecheck --silent
        npm run test:js -- \
            --no-file-parallelism \
            --passWithNoTests \
            "${JS_TEST_EXCLUDES[@]}" \
            "${setup_js_files[@]}"
    )
}

test_eslint_factory() {
    (cd eslint-factory && npm test)
}

test_go_packages() {
    make --no-print-directory test-unit BASE_REF="$BASE_REF"
}

if [ "${#go_packages[@]}" -gt 0 ]; then
    start_job "Go lint (${#go_packages[@]} package(s))" lint_go_packages
fi
if [ "${#setup_js_files[@]}" -gt 0 ] || [ "${#eslint_factory_files[@]}" -gt 0 ]; then
    start_job "JavaScript lint" lint_javascript
fi
if [ "${#action_shell_files[@]}" -gt 0 ]; then
    start_job "action shell lint" lint_action_shell_files
fi
if [ "$model_alias_validation_required" -eq 1 ]; then
    start_job "model alias validation" make --no-print-directory validate-model-alias-chains
fi
start_job "schema freshness check" make --no-print-directory check-stale-schema-binary

if [ "$WITH_TESTS" -eq 1 ]; then
    start_job "impacted Go tests" test_go_packages
    if [ "${#setup_js_files[@]}" -gt 0 ]; then
        start_job "impacted setup JavaScript tests" test_setup_javascript
    fi
    if [ "${#eslint_factory_files[@]}" -gt 0 ]; then
        start_job "eslint-factory tests" test_eslint_factory
    fi
fi

wait_for_jobs

if [ "$workflow_drift_required" -eq 1 ]; then
    make --no-print-directory check-workflow-drift
else
    echo "No workflow source or compiler changes; skipping full workflow recompilation."
fi

if [ "$WITH_TESTS" -eq 1 ]; then
    echo "Pre-PR validation passed (changed files formatted and linted, impacted tests pass). Safe to call report_progress."
else
    echo "Pre-PR validation passed (changed files formatted and linted). Safe to call report_progress."
fi
