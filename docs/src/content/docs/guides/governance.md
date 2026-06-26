---
title: Governance
description: Set and manage gh-aw environment defaults across enterprise, organization, and repository scopes with gh aw env.
---

# Governance

Use governance defaults when you want consistent model and
guardrail behavior across many repositories without editing
every workflow file.

This guide shows how to manage these defaults with
`gh aw env` and how values percolate from enterprise to
organization to repository scope.

## What `gh aw env` manages

`gh aw env` manages `GH_AW_DEFAULT_*` variables as GitHub
Actions variables at one scope:

- repository (`--scope repo`)
- organization (`--scope org`)
- enterprise (`--scope ent`)

The command uses a YAML file with `default_` keys.

```yaml title="defaults.yml"
default_max_ai_credits: "5M"
default_max_daily_ai_credits: "15M"
default_max_turns: "12"
default_timeout_minutes: "30"
default_model_copilot: "gpt-5-mini"
default_model_claude: "claude-haiku-4-5"
default_model_codex: "gpt-5.4-mini"
default_detection_model: "gpt-5.5-mini"
default_utc: "-08:00" # UTC offset for rendered CLI timestamps
```

> [!NOTE]
> Set a key to `null` (or omit it) to delete that variable
> from the selected scope during `gh aw env update`.

## Export current defaults

Start by exporting current values from the target scope.

```bash
gh aw env get ent-defaults.yml --scope ent --enterprise MY_ENT
gh aw env get org-defaults.yml --scope org --org MY_ORG
gh aw env get repo-defaults.yml --scope repo --repo OWNER/REPO
```

## Apply defaults safely

After editing the YAML file, preview and apply the change.

```bash
gh aw env update org-defaults.yml --scope org --org MY_ORG --dry-run
gh aw env update org-defaults.yml --scope org --org MY_ORG
```

Use `--yes` in automation to skip the interactive
confirmation prompt.

```bash
gh aw env update org-defaults.yml --scope org --org MY_ORG --yes
```

## Governance rollout pattern

Use a layered rollout to make defaults percolate down:

1. Set enterprise baseline defaults.
2. Set organization defaults only where needed.
3. Set repository defaults only for true exceptions.
4. Keep workflow frontmatter overrides rare and explicit.

This keeps most repositories aligned while still allowing
targeted exceptions.

## How percolation and precedence work

For values resolved from GitHub Actions `vars.*`, the most
specific scope wins:

1. workflow frontmatter value (if set)
2. repository variable
3. organization variable
4. enterprise variable
5. built-in compiler fallback

Examples using this runtime path include
`GH_AW_DEFAULT_MAX_AI_CREDITS`,
`GH_AW_DEFAULT_MAX_DAILY_AI_CREDITS`,
`GH_AW_DEFAULT_DETECTION_MAX_AI_CREDITS`,
and `GH_AW_DEFAULT_MODEL_*`.

> [!IMPORTANT]
> Some defaults are compile-time values.
> `GH_AW_DEFAULT_MAX_TURNS`,
> `GH_AW_DEFAULT_TIMEOUT_MINUTES`,
> `GH_AW_DEFAULT_MAX_TURN_CACHE_MISSES`,
> `GH_AW_DEFAULT_DETECTION_MODEL`, and
> `GH_AW_DEFAULT_UTC` are read by the compiler process.
> If compilation runs in CI, pass these values into the
> compile step environment.

```yaml title=".github/workflows/compile.yml"
jobs:
  compile:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v5
      - name: Compile workflows
        env:
          GH_AW_DEFAULT_MAX_TURNS: ${{ vars.GH_AW_DEFAULT_MAX_TURNS }}
          GH_AW_DEFAULT_TIMEOUT_MINUTES: ${{ vars.GH_AW_DEFAULT_TIMEOUT_MINUTES }}
          GH_AW_DEFAULT_MAX_TURN_CACHE_MISSES: ${{ vars.GH_AW_DEFAULT_MAX_TURN_CACHE_MISSES }}
          GH_AW_DEFAULT_DETECTION_MODEL: ${{ vars.GH_AW_DEFAULT_DETECTION_MODEL }}
          GH_AW_DEFAULT_UTC: ${{ vars.GH_AW_DEFAULT_UTC }}
        run: gh aw compile
```

## Runtime Policy Variables

Policy variables (`GH_AW_POLICY_*`) complement the default variables
managed by `gh aw env`. Where `GH_AW_DEFAULT_*` variables tune numeric and
model settings, policy variables enforce capability gates: a single boolean
value permits or refuses a specific behavior at runtime — without
recompiling any workflow.

Like default variables, policy variables are set as GitHub Actions variables
at repository, organization, or enterprise scope and picked up automatically
at workflow runtime through `vars.*`.

### Disabling `create-pull-request` org-wide

`GH_AW_POLICY_ALLOW_CREATE_PULL_REQUEST` controls whether agentic workflows
are allowed to open pull requests. Set it to `"false"` to prevent any
workflow from creating PRs across every repository in an organization or
enterprise:

```bash
gh variable set GH_AW_POLICY_ALLOW_CREATE_PULL_REQUEST \
  --org my-org --body "false"
```

When the policy is active, the safe-outputs server refuses to start for
any workflow that has `safe-outputs.create-pull-request` configured:

```text
create-pull-request is disabled by runtime policy: GH_AW_POLICY_ALLOW_CREATE_PULL_REQUEST=false.
Remove safe-outputs.create-pull-request or set GH_AW_POLICY_ALLOW_CREATE_PULL_REQUEST=true.
```

Any other value — including unset — leaves the tool enabled. To lift the
restriction at a specific repository scope or re-enable it org-wide, set
the variable to `"true"` or delete it:

```bash
# Re-enable for the whole org
gh variable delete GH_AW_POLICY_ALLOW_CREATE_PULL_REQUEST --org my-org

# Override at repository scope only (most-specific-wins)
gh variable set GH_AW_POLICY_ALLOW_CREATE_PULL_REQUEST \
  --repo owner/repo --body "true"
```

See [Runtime Policy Variables](/gh-aw/reference/environment-variables/#runtime-policy-variables)
for the complete list of `GH_AW_POLICY_*` variables.

---

## Troubleshooting

If `gh aw env update` fails validation:

- use positive integers for `default_max_turns`,
  `default_timeout_minutes`,
  `default_max_turn_cache_misses`
- use non-zero integers for
  `default_max_ai_credits`,
  `default_max_daily_ai_credits`,
  `default_detection_max_ai_credits`
- use a numeric UTC offset for `default_utc`
  (for example `+00:00` or `-08:00`)
- remove unknown YAML keys

## Related reference

- [Environment Variables](/gh-aw/reference/environment-variables/)
- [Compiler Enterprise Environment Controls](/gh-aw/reference/compiler-enterprise-environment-controls/)
- [Cost Management](/gh-aw/reference/cost-management/)
- [Using at Scale in Organizations](/gh-aw/guides/using-at-scale/)
