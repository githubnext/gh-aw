# ADR-35946: Manifest-First Skill Ordering and Recursive Skill Folder Copy

**Date**: 2026-05-30
**Status**: Draft
**Deciders**: Unknown

---

## Part 1 — Narrative (Human-Friendly)

### Context

[ADR-35778](35778-extend-aw-yml-for-skills-and-agents.md) added first-class skill distribution to the `aw.yml` package manifest, but the resolver used an *either/or* model: when `skills:` was present it used the explicit list, and otherwise it auto-scanned `skills/`. This left two gaps that surfaced as install bugs. First, ordering was non-deterministic relative to the manifest — a package could not guarantee that skills listed in `skills:` resolved before any extras discovered on disk. Second, the installer copied only the top-level files of each skill folder, so any nested subdirectory (e.g. `scripts/`, `prompts/`, `scripts/helpers/`) was silently dropped, breaking skills whose `SKILL.md` references helper scripts by relative path. Both gaps degrade a package author's ability to ship a complete, ready-to-run skill.

### Decision

We will resolve manifest-listed skills first and then **always** auto-scan `skills/`, appending only those skill folders not already covered by the manifest (deduplicated by skill name). We will also copy each skill folder **recursively**, preserving its full subdirectory structure on disk rather than flattening or dropping nested files. The relative path of each file within its skill directory is reconstructed using the `/<skill-name>/` path component as a boundary so that nesting is reproduced faithfully at the install destination.

### Alternatives Considered

#### Alternative 1: Keep the either/or model and require explicit listing for ordering control

We could preserve the original "explicit list OR auto-scan" behavior and tell authors to enumerate every skill in `skills:` when they care about order. Rejected because it reintroduces the manifest boilerplate ADR-35778 deliberately avoided and drifts out of sync as a package gains skills — a newly added skill folder would be invisible unless the author remembered to also list it. Manifest-first-then-append gives deterministic ordering *and* keeps zero-config discovery of extras.

#### Alternative 2: Flatten nested skill files into the skill root

Instead of preserving subdirectories, we could copy every nested file directly into `<skill-name>/`, discarding intermediate path segments. Rejected because skills reference their helpers by relative path (`scripts/query.sh`); flattening would break those references and risk filename collisions between files in different subdirectories.

#### Alternative 3: Keep non-recursive copy and constrain skills to flat folders

We could leave the copy logic untouched and declare that a skill folder must contain only top-level files. Rejected as too restrictive: real skills in this repository already use `scripts/` and similar subdirectories, so a flat-only rule would make them un-shippable via packages.

### Consequences

#### Positive
- Skill ordering is deterministic: manifest skills always precede auto-scanned extras, so a package author controls precedence by listing.
- Entire skill folders install intact, including nested `scripts/`, `prompts/`, and deeper helper directories, so skills that depend on relative-path assets work after `gh aw add`.
- Auto-scan still runs even when `skills:` is present, so additional on-disk skills are no longer hidden by the presence of a manifest list; name-based deduplication prevents a manifest skill from being installed twice.

#### Negative
- Recursive enumeration via the GitHub Contents API issues one additional API call per subdirectory, increasing latency and rate-limit pressure for deeply nested skills (mitigated by a git-clone `ls-tree -r` fallback).
- This relaxes the "single-level nesting / direct child only" contract asserted in ADR-35778's normative section; that ADR's nesting language is now partially superseded and should be reconciled.
- Auto-scan now always executes, adding directory-listing and `SKILL.md` marker calls even for packages that fully specify `skills:`, where the previous code skipped scanning entirely.

#### Neutral
- A new injectable seam `listPackageDirFilesRecursivelyForHost` is introduced so tests can mock recursive listing independently of the existing non-recursive helper (agents still use the non-recursive path).
- Relative-path reconstruction relies on locating the `/<skill-name>/` path component within the source path, falling back to `filepath.Base` when the boundary is absent.
- The recursive walk is exposed publicly as `parser.ListDirAllFilesRecursivelyForHost`, paralleling the existing `ListDirAllFilesForHost` helper.

---

## Part 2 — Normative Specification (RFC 2119)

> The key words **MUST**, **MUST NOT**, **REQUIRED**, **SHALL**, **SHALL NOT**, **SHOULD**, **SHOULD NOT**, **RECOMMENDED**, **MAY**, and **OPTIONAL** in this section are to be interpreted as described in [RFC 2119](https://www.rfc-editor.org/rfc/rfc2119).

### Skill Resolution Order

1. The resolver **MUST** resolve manifest-listed (`skills:`) skill directories before any auto-scanned skill directories.
2. The resolver **MUST** auto-scan the `skills/` directory even when the manifest provides an explicit `skills:` list, appending any discovered skill folder not already covered by the manifest.
3. The resolver **MUST** deduplicate skills by skill name so that a skill present in both the manifest and the auto-scan is resolved exactly once, retaining its manifest-first position.
4. A manifest-listed skill directory **MUST** be validated against its `SKILL.md` marker, and a missing marker **MUST** produce a warning rather than aborting resolution.

### Recursive Folder Copy

1. The installer **MUST** copy every file under a resolved skill folder at any nesting depth, not only the folder's top-level files.
2. The installer **MUST** preserve each file's path relative to the skill directory when writing it to the destination, recreating intermediate subdirectories as needed.
3. The installer **MUST NOT** flatten nested skill files into the skill root or otherwise discard their subdirectory structure.
4. Relative-path reconstruction **MUST** locate the skill name as a complete `/<skill-name>/` path component so that a subdirectory whose name coincides with the skill name does not corrupt the computed relative path; when no such component is found, the installer **SHOULD** fall back to the file's base name.

### Remote Listing

1. Recursive remote listing **MUST** attempt the GitHub Contents API first and **MUST** fall back to a git clone with `ls-tree -r` on authentication or client-creation failure.
2. The git fallback **MUST** return files at any depth under the target directory, not only direct children.

### Conformance

An implementation is considered conformant with this ADR if it satisfies all **MUST** and **MUST NOT** requirements above. Failure to meet any **MUST** or **MUST NOT** requirement constitutes non-conformance.

---

*This is a DRAFT ADR generated by the [Design Decision Gate](https://github.com/github/gh-aw/actions/runs/26687535424) workflow. The PR author must review, complete, and finalize this document before the PR can merge.*
