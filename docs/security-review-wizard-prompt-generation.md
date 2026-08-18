# Security Review: Wizard Prompt Generation and Rendering

This document records a focused security/design review for issue #53520 (epic #53498, "AW wizard in docs"). Scope was limited to user-supplied text flowing into generated prompt content, client-side rendering of that content, and trust boundaries around the shared wizard JSON catalog.

## Summary of scope

Reviewed files:

- `docs/src/components/Wizard.astro`
- `docs/src/lib/wizard/model.ts`
- `docs/src/lib/wizard/validation.ts`
- `docs/src/lib/wizard/model.test.js`
- `docs/tests/wizard.spec.ts`
- Supporting schema/spec artifacts used to define future reuse expectations

Primary questions:

1. How free-text task description input is handled before display and copy/paste.
2. Whether generated prompt content is rendered through safe text sinks or unsafe HTML sinks.
3. Whether the shared `wizard-data-model.json` is treated as trusted static repo content today, and what guardrails future CLI/server consumers will need if they ingest equivalent JSON from less-trusted sources.

## Findings

### 1. Free-text task description is safe for browser rendering, but prompt-injection risk remains by design

- **Severity**: Low
- **Location**: `docs/src/components/Wizard.astro` (`renderTaskDescriptionField()`, `buildPromptText()`, preview/final prompt rendering)
- **Description**: The wizard takes arbitrary user text from the task-description textarea and appends it verbatim into the generated prompt string. That is the intended product behavior, but it also means a user can paste instruction-conflicting or manipulative content into the prompt that they later hand to another AI system. This is not a browser XSS issue; it is a downstream prompt-integrity issue. The current implementation does not distinguish trusted scaffold text from untrusted user-authored task details, and it does not set any size limits on the textarea input.
- **Current mitigation**:
  - Rendering uses `textContent`/`pre.textContent`, so the free text is inserted as literal text rather than HTML.
  - Clipboard copy uses `navigator.clipboard.writeText(promptText)`, which preserves plain text rather than rich HTML.
  - The summary view truncates the visible task description preview to 80 characters, reducing accidental UI blow-up.
- **Recommended guardrail**:
  - Before merge, explicitly document that task details are untrusted user input and are copied into the generated prompt verbatim.
  - Add a user-facing note near the wizard or in docs warning not to paste secrets and to review task details before sending the prompt to another model.
  - Add a reasonable maximum length for free-text task details to avoid oversized prompt generation and denial-of-usability cases.
  - Future consumers should treat the task-details field as data only and MUST NOT feed it into any code execution, shell execution, template evaluation, or workflow file emission without context-appropriate escaping/validation.

### 2. Client-side prompt rendering currently avoids unsafe HTML/XSS sinks

- **Severity**: Info
- **Location**: `docs/src/components/Wizard.astro`
- **Description**: The reviewed client-side rendering path uses DOM APIs that assign plain text (`textContent`, `pre.textContent`) for the live preview, summary, step labels, and final prompt display. No `innerHTML`, `outerHTML`, `insertAdjacentHTML`, DOM parsing, scriptable URL construction, or event-handler attribute sinks were found in the wizard component.
- **Current mitigation**:
  - Prompt preview uses `preview.textContent = buildPromptText()`.
  - Final prompt view uses `pre.textContent = promptText`.
  - Option labels/help and summary rows are all rendered through `textContent`.
  - The copied artifact is plain text via the Clipboard API, not rendered HTML.
- **Recommended guardrail**:
  - Preserve the invariant that wizard-derived content must only be rendered through text sinks unless there is a separately reviewed sanitizer.
  - Add a brief engineering note in docs/review context stating that future rich rendering of prompt/model content must not switch to `innerHTML` or Markdown-to-HTML rendering without explicit sanitization review.

### 3. Shared wizard JSON model is trusted static repo content today, but future non-docs consumers need stricter trust-boundary rules

- **Severity**: Medium
- **Location**: `docs/src/lib/wizard/model.ts`, `docs/src/lib/wizard/validation.ts`, `docs/src/data/wizard-data-model.json`, `docs/public/schemas/wizard-data-model.schema.json`, `docs/src/content/docs/specs/wizard-data-model-specification.md`
- **Description**: In the current docs implementation, `wizard-data-model.json` is checked-in repository content imported at build/runtime and validated for basic shape. In that narrow context it is effectively trusted content controlled by repository maintainers. However, the schema/spec explicitly position the model for reuse by a future CLI consumer, and the schema allows free-form `frontmatter` objects that a later generator may merge "as-is." If a future CLI/server implementation accepts equivalent JSON from external sources, the trust boundary changes materially: malformed, adversarial, oversized, or semantically dangerous catalog data could influence generated workflow content, frontmatter merges, or output routing. The current TypeScript validator checks presence/type of core fields but does not enforce `additionalProperties: false`, uniqueness of IDs, referential integrity, string length caps, or defensive rejection of suspicious keys in free-form objects.
- **Current mitigation**:
  - The docs runtime imports a checked-in JSON file rather than fetching remote/user-supplied model data.
  - `validateModel()` ensures the top-level object shape and required arrays/fields exist.
  - The published JSON Schema is stricter than the runtime validator and already describes `additionalProperties: false` for many structures.
  - The spec says the model is reusable by a future CLI consumer, which creates a natural place to record stronger trust assumptions.
- **Recommended guardrail**:
  - Before merge, document that `docs/src/data/wizard-data-model.json` is trusted static repository content for the docs wizard only; future consumers must not assume equivalent JSON is trusted.
  - Document that any CLI/server consumer loading model data from disk, network, or plugins must validate against the full JSON Schema, enforce size limits, and treat `frontmatter` fragments as untrusted structured data until merged under an allowlisted policy.
  - Future generators should avoid prototype-polluting or surprising merge behavior when handling free-form objects such as `frontmatter`; prefer schema validation plus safe deep-merge logic that rejects dangerous keys and unknown top-level fields.
  - Consider strengthening runtime validation in a follow-up hardening issue if/when the shared model is consumed outside the current checked-in docs path.

### 4. Runtime validation is adequate for current docs use, but missing abuse-resistance limits for future reuse

- **Severity**: Low
- **Location**: `docs/src/lib/wizard/validation.ts`, `docs/src/lib/wizard/model.test.js`
- **Description**: The current validator is aimed at catching malformed checked-in data early, not defending a hostile ingestion boundary. It does not cap string lengths, array sizes, or aggregate model size; does not verify uniqueness of `id` values; and does not ensure goal references only point to defined trigger/destination IDs. That is acceptable for a small, reviewed docs data file, but insufficient if the same validation entry point is reused for external model ingestion later.
- **Current mitigation**:
  - Module-load validation prevents obviously malformed checked-in JSON from silently reaching the docs build.
  - Unit tests cover several malformed-shape failures.
- **Recommended guardrail**:
  - Before merge, record in docs that `validateModel()` is a shape validator for checked-in docs content, not a full security boundary for untrusted JSON.
  - For future reuse, require caps on string/array sizes, referential-integrity checks, and duplicate-ID rejection before treating the model as acceptable input.

## Guardrails required before merge

- [ ] Document that task-description text is untrusted user input, is copied verbatim into the generated prompt, and should not include secrets.
- [ ] Document that wizard-generated prompt text is plain text only and must not be rendered with `innerHTML`/unsafe HTML sinks without sanitization review.
- [ ] Document that `docs/src/data/wizard-data-model.json` is trusted static repo content in the docs implementation, but future CLI/server/plugin consumers must treat equivalent model JSON and `frontmatter` fragments as untrusted input.
- [ ] Add or track a reasonable maximum length for free-text task details before this ships broadly, to prevent oversized prompt output and reduce abuse potential.
- [ ] Record that `validateModel()` is not, by itself, a sufficient security boundary for externally supplied wizard model data.

## Not in scope

This was a focused hardening/documentation pass, not a functional QA or product-behavior review. It did not attempt broad UX validation, step-flow correctness, schema evolution design, or full end-to-end testing beyond reviewing the existing coverage relevant to rendering and model validation.
