# ADR-70840: Serialize Job Fields as Keyed YAML and Align Merged Nodes by Key Column

**Date**: 2026-05-13
**Status**: Draft
**Deciders**: pelikhan, Copilot coding agent

---

### Context

Custom jobs and safe-output jobs that use more than one `runs-on` label compiled with extra indentation after the first label (issue #70811). The output was not the intended label list, so jobs matched the wrong runners and stayed queued. `formatIndentedYAMLField` in `pkg/workflow/compiler_custom_job_execution.go` marshalled only the field value, wrote `fieldName:` by hand, and then added a fixed six-space indent to every marshalled line. This broke the relative indentation of sequence items. A related problem existed in the Actions/AW source-AST merge path (`pkg/workflow/actions/source_merge.go`, `source_sequence.go`). There, the column used to re-indent a replacement node came from the node's own token. For block mappings, that token points at the first colon, not the first key. The fix for this colon-vs-key case was written inline only in `mergeSourceMapping` and `mergeSourceSequence`.

### Decision

We will have the YAML encoder serialize the job field together with its key (`map[string]any{fieldName: value}`). We will then indent only the continuation lines with the job-level offset, so the encoder alone controls the nesting of sequence and mapping items. In the source-AST merge path, we will compute a node's indentation column with one shared helper, `sourceNodeColumn`. The helper uses the smallest column among the node token and its mapping keys. Every place that replaces or merges a node (`replaceSourceNode`, `mergeSourceExistingFields`, `mergeSourceMapping`, `mergeSourceSequence`) will use it. The main reason is correctness: indentation must come from one consistent source and not from hand-written prefixes.

### Alternatives Considered

#### Alternative 1: Adjust the hand-written prefix in `formatIndentedYAMLField`

We could have kept marshalling the bare value and tuned the fixed `"      "` prefix, or handled the first line separately. This would be a smaller diff. However, the field's indentation would still be built separately from the encoder's output. It would stay fragile for nested sequences, mappings (for example `runs-on: {group, labels}`), and future encoder options such as `IndentSequence`. That is the kind of mismatch that caused this bug.

#### Alternative 2: Emit custom jobs through a fully structured YAML tree

We could have built the whole job as a YAML value tree and serialized it once, with no string concatenation. This would remove the whole class of indentation bugs. It would also be a much larger refactor of custom/safe job generation than this regression fix needs.

#### Alternative 3: Fix only the call sites that show the colon-vs-key problem

In the source-AST merge, we could have patched `replaceSourceNode` and `mergeSourceExistingFields` inline, as `mergeSourceMapping` and `mergeSourceSequence` already did. That would leave four copies of the same column logic, which could drift apart.

### Consequences

#### Positive
- Multi-label `runs-on` values (and other sequence or mapping job fields) keep their sibling indentation in compiled custom and safe-output jobs.
- Column calculation for AST merges is defined once in `sourceNodeColumn`, so mapping replacements in every merge path follow the same rule.
- New tests check exact indentation and round-trip preservation of scalar, label-sequence, and group/labels runner forms.

#### Negative
- `formatIndentedYAMLField` now assumes the encoder puts the key on the first line and that continuation lines need exactly the job-level offset (four spaces). Changing the job nesting depth requires updating this constant.
- The output still depends on encoder options (`DefaultMarshalOptions` plus `IndentSequence(true)`). Changing those options globally changes the compiled lock-file layout.

#### Neutral
- Expected outputs in existing tests (for example `safe_jobs_test.go`) changed to match the corrected indentation. Recompiled lock files may show whitespace-only diffs for affected jobs.
- Package docs for `actions` and `aw` now state that runner selections keep their native scalar, sequence, and mapping forms, including unevaluated expression strings.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
