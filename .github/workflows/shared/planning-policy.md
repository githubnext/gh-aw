# Planning Policy

## Purpose
Planning policy controls how Squad's planning workflow behaves for a given repository. 
Customers configure policy in their issue body or `.squad/planning-policy.md` file.
When no policy is specified, sensible defaults apply.

## Policy Schema

### Artifact Limits
| Setting | Default | Description |
|---------|---------|-------------|
| `max_issues` | 20 | Maximum issues created per activation |
| `max_milestones` | 5 | Maximum milestones per program plan |
| `max_epics` | 10 | Maximum epics per program plan |
| `max_tasks_per_epic` | 8 | Maximum implementation tasks per epic |

### Sizing
| Setting | Default | Description |
|---------|---------|-------------|
| `sizing_scale` | `XS,S,M,L,XL` | Valid size values |
| `max_task_size` | `L` | Tasks larger than this must be split |
| `sizing_method` | `relative` | `relative` (story points-like) or `none` (skip sizing) |

### Hierarchy
| Setting | Default | Description |
|---------|---------|-------------|
| `require_milestones` | `true` | Epics must be assigned to milestones |
| `require_acceptance_criteria` | `true` | All items must have acceptance criteria |
| `require_agent_assignment` | `false` | Tasks must have an assigned agent |
| `min_stories_per_epic` | 1 | Minimum user stories per epic |

### GitHub Representation
| Setting | Default | Description |
|---------|---------|-------------|
| `use_github_milestones` | `true` | Create GitHub milestones (vs. just documenting them) |
| `use_sub_issues` | `true` | Use native sub-issue links when available |
| `use_project_fields` | `false` | Write to GitHub Project fields (requires project) |
| `label_strategy` | `squad` | Label prefix for created issues |
| `size_representation` | `body` | `body` (in issue body), `label` (as labels), `project_field` |

### Validation Strictness
| Setting | Default | Description |
|---------|---------|-------------|
| `strict_traceability` | `true` | Every impl task must trace to a program item |
| `allow_oversized_with_justification` | `false` | Allow L+ tasks if rationale provided |
| `require_all_decisions_resolved` | `true` | Block acceptance until all decisions resolved |
| `cycle_detection` | `error` | `error` (block), `warn` (advisory), `skip` |

## Profiles

Profiles are named bundles of settings for common use cases.

### `default` — Balanced
All defaults as specified above. Good for most projects.

### `lean` — Minimal Process
```
max_issues: 10
require_milestones: false
require_acceptance_criteria: false
require_agent_assignment: false
sizing_method: none
strict_traceability: false
```

### `enterprise` — Maximum Rigor
```
max_issues: 50
require_milestones: true
require_acceptance_criteria: true
require_agent_assignment: true
strict_traceability: true
allow_oversized_with_justification: false
require_all_decisions_resolved: true
use_github_milestones: true
use_project_fields: true
```

### `spike` — Exploration/Prototype
```
max_issues: 5
max_epics: 3
require_milestones: false
require_acceptance_criteria: false
sizing_method: none
strict_traceability: false
cycle_detection: warn
```

## How Policy is Resolved (precedence order)

1. **Issue body** — HTML comment `<!-- squad-policy: lean -->` or inline settings
2. **Repository file** — `.squad/planning-policy.md` with YAML frontmatter
3. **Profile name** — Matches a built-in or custom profile
4. **Default** — All defaults apply

## Custom Profiles

Users can define custom profiles in `.squad/planning-policy.md`:

```yaml
---
profile: custom
settings:
  max_issues: 30
  require_milestones: true
  sizing_method: relative
  max_task_size: M
---
```

## Policy in Issue Body

Embed policy inline for per-issue customization:

```markdown
<!-- squad-policy: lean -->
```

Or override specific settings:

```markdown
<!-- squad-policy: default -->
<!-- squad-setting: max_issues=30, require_milestones=false -->
```
