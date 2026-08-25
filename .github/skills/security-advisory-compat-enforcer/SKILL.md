---
name: security-advisory-compat-enforcer
description: Review a GitHub security advisory and safely update .github/aw/compat.json with evidence-backed version enforcement.
---

# Security Advisory Compatibility Enforcer

Use this skill to translate a GitHub repository security advisory into the
smallest safe update to `.github/aw/compat.json`.

## Required inputs

Record the repository, GHSA identifier, requested enforcement policy, and any
patched version supplied by the user. Treat a user-supplied version as a target
to verify, not as evidence for advisory details.

## Evidence retrieval

1. Attempt authenticated retrieval first:
   `gh api repos/OWNER/REPO/security-advisories/GHSA-ID`.
2. If authentication is unavailable or access fails, try the public global
   advisory endpoint: `gh api advisories/GHSA-ID` (or its equivalent REST URL).
3. Record the endpoint and outcome of each attempt. If both sources are
   inaccessible, state that explicitly. Never infer or invent the affected
   package, vulnerable range, severity, CVE, publication state, or patched
   version.
4. Verify the proposed patched version independently. Prefer a published
   repository release or tag, then package-registry metadata when applicable.
   Cite the exact URL or command result. Do not update compatibility policy if
   the target cannot be verified, unless the user explicitly directs use of a
   supplied target despite inaccessible advisory metadata; document that
   limitation without converting it into fabricated advisory evidence.

## Choose the correct policy field

- `minimumVersion` is a hard floor: activation fails for every compiler version
  below it. Change it only when the requested remediation is a universal
  minimum-version enforcement.
- `blockedVersions` rejects listed versions exactly. Change it only when
  evidence or explicit instructions identify exact versions to deny and a
  continuous minimum floor would be inaccurate.
- `minRecommendedVersion` only warns below the value. Change it only for an
  explicitly requested recommendation, never as a substitute for enforcement.
- Agent rows under `agent-compat-v1` select compatible agent versions; they are
  unrelated to compiler security enforcement unless separate evidence requires
  an agent compatibility change.

## Safe edit procedure

1. Parse the current JSON and record all four policy areas above.
2. Compare semantic versions numerically. A minimum is monotonic: never lower a
   non-empty `minimumVersion` or `minRecommendedVersion`. Stop and report a
   requested downgrade rather than applying it.
3. Make the narrowest evidence-backed edit. Preserve `blockedVersions`,
   `minRecommendedVersion`, every `agent-compat-v1` row, key ordering, and
   formatting unless the selected policy specifically requires changing them.
4. Review the final diff and reject unrelated changes.

## Required validation

Before reporting completion:

1. Parse `.github/aw/compat.json` as JSON.
2. Validate it against `.github/aw/compat.schema.json` using the repository's
   schema validator.
3. Exercise the runtime policy semantics with versions immediately below, at,
   and above the changed boundary; confirm only the intended hard-fail, warning,
   or exact-block behavior changed.
4. Confirm semantic-version monotonicity and byte-for-byte preservation of
   unrelated policy fields and agent rows.

Do not claim validation that was not run. If repository constraints prohibit a
required check, report it as outstanding.

## Report

Cite advisory retrieval attempts and patched-version verification. State which
field changed, old and new values, why that policy is correct, which fields were
preserved, and the schema/runtime validation results. Clearly separate verified
facts, user-provided inputs, and unavailable advisory details.
