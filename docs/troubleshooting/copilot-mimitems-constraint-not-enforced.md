# Copilot CLI Does Not Enforce `minItems` JSON Schema Constraint

## Issue Description

**Error Context:**
The `remove_labels` safe output tool in workflow run [21215591937](https://github.com/githubnext/gh-aw/actions/runs/21215591937/job/61036204152#step:6:1) allowed the Copilot agent to generate a tool call with an empty labels array `[]`, despite having `"minItems": 1` in the JSON schema definition.

**Expected Behavior:**
The `minItems: 1` constraint should prevent the AI agent from generating a tool call with an empty array.

**Actual Behavior:**
The Copilot agent successfully generated a tool call with `labels: []`, which was then rejected by the JavaScript handler during execution.

## Root Cause

**Copilot CLI does not enforce the `minItems` JSON Schema constraint** when generating tool calls for MCP tools.

### Evidence

1. **Schema Definition** (`actions/setup/js/safe_outputs_tools.json`, lines 296-317):
   ```json
   {
     "name": "remove_labels",
     "inputSchema": {
       "type": "object",
       "properties": {
         "labels": {
           "type": "array",
           "items": { "type": "string" },
           "minItems": 1,  // ← Not enforced by Copilot CLI
           "description": "Label names to remove ... At least one label must be provided."
         }
       },
       "required": ["labels"],
       "additionalProperties": false
     }
   }
   ```

2. **Workflow Run Logs**:
   ```
   2026-01-21T15:40:11.9810801Z Requested labels to remove: []
   2026-01-21T15:40:11.9819595Z No labels provided. Please provide at least one label...
   2026-01-21T15:40:11.9846468Z ##[error]✗ Message 2 (remove_labels) failed: No labels provided...
   ```

3. **Uniqueness of `minItems` Usage**:
   - Only `remove_labels` uses `minItems` constraint in the entire tools schema
   - No other tools use `minItems`, suggesting it's not a standard pattern
   - Query: `jq '.[] | select(.inputSchema.properties | to_entries[] | select(.value.minItems != null))' safe_outputs_tools.json` returns only `remove_labels`

4. **Previous Fix Attempt**:
   - Commit [4d51699](https://github.com/githubnext/gh-aw/commit/4d516999ed02de121cf04112396fab9431c5a0f4) added `minItems: 1` as a fix
   - Testing in workflow run 21215591937 proved the constraint is not enforced
   - The constraint exists in the schema but has no effect on tool call generation

## Why `minItems` is Not Enforced

Copilot CLI appears to use a **simplified JSON Schema validator** for MCP tool schemas that:
- ✅ Validates `type` (object, array, string, number)
- ✅ Validates `required` fields
- ✅ Validates `additionalProperties`
- ❌ Does NOT validate array constraints (`minItems`, `maxItems`)
- ❌ Does NOT validate string constraints (`minLength`, `maxLength`, `pattern`)
- ❌ Does NOT validate number constraints (`minimum`, `maximum`)

This is consistent with the [documented Copilot CLI schema validation issues](./copilot-schema-validation-deep-analysis.md), which show intermittent problems with schema enforcement.

## Current Enforcement Mechanism

### JavaScript Handler Validation (Effective)

The **actual enforcement** happens in the JavaScript handler (`actions/setup/js/remove_labels.cjs`, lines 67-79):

```javascript
// If no labels provided, return a helpful message
if (!requestedLabels || requestedLabels.length === 0) {
  let errorMessage = "No labels provided. Please provide at least one label from";
  if (allowedLabels.length > 0) {
    errorMessage += ` the allowed list: ${JSON.stringify(allowedLabels)}`;
  } else {
    errorMessage += " the issue/PR's current labels";
  }
  core.info(errorMessage);
  return {
    success: false,
    error: errorMessage,
  };
}
```

This validation:
- ✅ **Works correctly** - rejects empty arrays
- ✅ **Provides helpful error messages** - shows allowed labels when configured
- ✅ **Runs during execution** - after the AI has already generated the tool call
- ❌ **Runs too late** - the invalid tool call has already been generated and sent

### Tool Description (Partially Effective)

The `labels` property description mentions the constraint:
```
"Label names to remove (e.g., ['bug', 'needs-triage']). Labels that don't exist on the item are silently skipped. At least one label must be provided."
```

This helps the AI understand the requirement but does not **enforce** it at schema validation time.

## Impact

### Severity: Low

- **Frequency**: Depends on AI behavior - not guaranteed to occur
- **Impact**: Tool call fails during execution with clear error message
- **Workaround**: JavaScript handler provides correct validation
- **User Experience**: Error is caught and reported; no data corruption

### Why This is Acceptable

1. **Defense in Depth**: The JavaScript handler provides a second layer of validation
2. **Clear Error Messages**: Users get actionable feedback when constraint is violated
3. **No Security Impact**: Empty array causes tool call to fail safely
4. **Documented Behavior**: This document explains the limitation

## Recommendations

### For Workflow Authors

1. ✅ **Rely on JavaScript handler validation** - it's the effective enforcement point
2. ✅ **Include constraints in descriptions** - helps the AI make better decisions
3. ✅ **Keep `minItems` in schema** - may be enforced in future Copilot CLI versions
4. ❌ **Don't assume schema constraints are enforced** - always validate in handlers

### For gh-aw Maintainers

1. ✅ **Keep `minItems: 1` in schema** - documents intent and enables future enforcement
2. ✅ **Document this limitation** - prevent confusion about expected behavior
3. ✅ **Maintain JavaScript validation** - the actual enforcement mechanism
4. ✅ **Monitor Copilot CLI updates** - schema constraint support may improve

### Future Improvements

If Copilot CLI adds support for `minItems` in the future:
- The existing schema constraint will automatically take effect
- JavaScript handler validation remains as defense-in-depth
- No code changes required in gh-aw

## Testing

To verify this behavior:

```bash
# 1. Check that minItems exists in schema
jq '.[] | select(.name == "remove_labels") | .inputSchema.properties.labels.minItems' \
  actions/setup/js/safe_outputs_tools.json
# Expected: 1

# 2. Check that only remove_labels uses minItems
jq '[.[] | select(.inputSchema.properties | to_entries[] | select(.value.minItems != null)) | .name]' \
  actions/setup/js/safe_outputs_tools.json
# Expected: ["remove_labels"]

# 3. Test with a workflow that tries to call remove_labels with empty array
# (See workflow run 21215591937 for example)
```

## Related Issues

- [Copilot Schema Validation Error](./copilot-schema-validation-error.md) - Intermittent schema validation bugs
- [Deep Analysis: Copilot Schema Validation](./copilot-schema-validation-deep-analysis.md) - Complete message flow analysis
- Workflow Run: https://github.com/githubnext/gh-aw/actions/runs/21215591937

## Status

- **Status**: Documented Known Limitation
- **Copilot CLI Version**: 0.0.384+ (all versions)
- **gh-aw Version**: All versions
- **Resolution**: Working as designed; JavaScript handler provides enforcement

## Conclusion

The `minItems: 1` JSON Schema constraint in the `remove_labels` tool is **not enforced by Copilot CLI** during tool call generation. This is a **known limitation** of Copilot CLI's schema validation, not a bug in gh-aw.

The **actual enforcement** happens in the JavaScript handler (`remove_labels.cjs`), which correctly validates and rejects empty label arrays. This provides defense-in-depth and ensures invalid tool calls are caught before execution.

**Recommendation**: Keep the `minItems` constraint in the schema for documentation and future compatibility, while relying on the JavaScript handler for actual enforcement.
