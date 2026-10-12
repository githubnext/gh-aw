---
private: true
emoji: "🧪"
description: Daily test workflow using Claude with custom safe-output job containing choice inputs
on:
  schedule:
    - cron: "daily around 12:00 on weekdays"  # ~12 PM UTC, weekdays only
  workflow_dispatch:
max-daily-ai-credits: 10000
permissions:
  contents: read
  issues: read
  pull-requests: read
tracker-id: daily-choice-test
engine: claude
strict: true
timeout-minutes: 10
experiments:
  emoji_density:
    variants: [minimal, heavy]
    description: "Effect of emoji density on readability of weekday choice explanations"
    hypothesis: "H0: no readability improvement. H1: heavy improves mean readability score by at least 0.5/5"
    metric: readability_score
    secondary_metrics: [run_duration_seconds, explanation_word_count, "eval:emoji_assignment_followed"]
    guardrail_metrics:
      - name: "eval:choice_correct"
        direction: max
        threshold: 1.0
      - name: missing_explanation_rate
        direction: min
        threshold: 0.0
      - name: failed_run_rate
        direction: min
        threshold: 0.0
    min_samples: 50
    start_date: "2026-10-12"
    issue: 67698
evals:
  include-assistant-text: true
  questions:
    - id: emoji_assignment_followed
      question: "Using the independent assignment in run_evidence, does the final paragraph contain zero emoji when assigned minimal, or exactly three emoji when assigned heavy, while spelling out the environment, test type, and weekday rationale in words? If the assignment is unavailable, answer UNKNOWN."
    - id: choice_correct
      question: "Using the UTC run date and structured_choice_correct in run_evidence, does exactly one test_environment declaration select staging/smoke on Monday, Wednesday, or Friday, or production/integration on Tuesday or Thursday, with an explanation that agrees with the declaration? If the deterministic check is UNKNOWN, answer UNKNOWN; do not infer the declaration from prose."
      structured_check:
        item_type: test_environment
        result_key: structured_choice_correct
        expected_by_utc_weekday:
          "1": { environment: staging, test_type: smoke }
          "2": { environment: production, test_type: integration }
          "3": { environment: staging, test_type: smoke }
          "4": { environment: production, test_type: integration }
          "5": { environment: staging, test_type: smoke }

network:
  allowed:
    - defaults

sandbox:
  agent:
    id: awf
tools:
  cli-proxy: true
  github:
    mode: local
    toolsets:
      - default

safe-outputs:
  staged: true
  jobs:
    test_environment:
      name: test_environment
      description: "A test job with choice input"
      runs-on: ubuntu-latest
      inputs:
        environment:
          description: "Target environment"
          required: true
          type: choice
          options: ["staging", "production"]
        test_type:
          description: "Type of test to run"
          required: true
          type: choice
          options: ["smoke", "integration", "e2e"]
      output: "Environment test completed successfully"
      steps:
        - name: Display Test Config
          run: |
            if [ -f "$GH_AW_AGENT_OUTPUT" ]; then
              ENVIRONMENT=$(cat "$GH_AW_AGENT_OUTPUT" | jq -r '.items[] | select(.type == "test_environment") | .environment')
              TEST_TYPE=$(cat "$GH_AW_AGENT_OUTPUT" | jq -r '.items[] | select(.type == "test_environment") | .test_type')
              {
                echo "### Test Configuration"
                echo "- Environment: $ENVIRONMENT"
                echo "- Test Type: $TEST_TYPE"
              } >> "$GITHUB_STEP_SUMMARY"
              echo "Test would be executed on $ENVIRONMENT environment with $TEST_TYPE tests"
            else
              echo "No agent output found"
            fi
imports:
  - shared/otlp.md
features:
  gh-aw-detection: true
---

# Daily Choice Type Test

This workflow tests the choice type functionality in safe-output jobs with Claude.

## Task

Use the `test_environment` tool to configure a test deployment. Choose:
1. An environment: staging or production
2. A test type: smoke, integration, or e2e

Make your selection based on the day of the week:
- Monday/Wednesday/Friday: Use "staging" environment with "smoke" tests
- Tuesday/Thursday: Use "production" environment with "integration" tests

Provide a brief explanation of why you chose this configuration.

### Output Format

Respond with one paragraph of 25–45 words explaining the chosen environment,
test type, and the run's UTC weekday rationale. Spell out all three facts.
{{#if experiments.emoji_density == "heavy" }}
Assigned presentation variant: heavy.
Include exactly three emoji: one adjacent to the environment, one adjacent
to the test type, and one adjacent to the weekday rationale.
{{else}}
Assigned presentation variant: minimal.
Use zero emoji in the explanation.
{{/if}}
Keep emoji out of test_environment tool arguments. Invoke the tool exactly once.