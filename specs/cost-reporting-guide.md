# Cost Reporting in gh-aw

## Understanding Token Costs

### How Costs are Calculated

The `EstimatedCost` field in workflow logs represents the **total cost in USD** for the entire workflow run, as reported by the AI engine's API. This is NOT a per-token cost.

For Claude models, different token types have different pricing:

| Token Type | Claude Sonnet 4 | Cost per 1M tokens | Cost per 1K tokens |
|------------|-----------------|--------------------|--------------------|
| Input tokens | Standard | $3.00 | $0.003 |
| Cache writes | cache_creation_input_tokens | $3.75 | $0.00375 |
| Cache reads | cache_read_input_tokens | $0.30 | $0.0003 |
| Output tokens | Standard | $15.00 | $0.015 |

### Common Misconception: Average Cost Per Token

❌ **Incorrect**: Dividing `total_cost_usd` by `total_tokens` to get "average cost per token"

This calculation is misleading because:
1. Different token types have vastly different pricing (output tokens cost 5x more than input tokens)
2. Cache reads are 10x cheaper than regular input tokens
3. The "average" will vary wildly depending on the mix of token types used

### Example

Given a workflow run with:
- 6 input tokens
- 78,962 cache write tokens  
- 0 cache read tokens
- 152 output tokens
- **Total**: 79,120 tokens
- **Total cost**: $0.2988

**Incorrect calculation**:
```
Average cost per token = $0.2988 / 79,120 = $0.00000378
Average cost per 1K tokens = $0.00000378 × 1,000 = $0.00378
```

This "average" of $0.00378/1K tokens is misleading because it's dominated by the cheap cache write tokens.

**Correct understanding**:
```
Cost breakdown:
- Input: 6 × $0.000003 = $0.000018
- Cache writes: 78,962 × $0.00000375 = $0.296108
- Cache reads: 0 × $0.0000003 = $0.000000
- Output: 152 × $0.000015 = $0.002280
Total: $0.298406 ≈ $0.2988 (API reported)
```

The API-reported cost of $0.2988 is accurate and accounts for the different pricing of each token type.

### Reporting Best Practices

When displaying cost metrics in reports:

1. **Always show total cost**, not "per token" averages
2. **If showing per-run averages**, make it clear it's "average cost per run", not "cost per token"
3. **If breaking down by token type**, show separate counts and costs for:
   - Input tokens
   - Cache write tokens
   - Cache read tokens  
   - Output tokens
4. **Avoid calculating** "average cost per 1K tokens" unless you can separate by token type

### Cost Data Sources

- **Source**: `total_cost_usd` field in Claude result logs (type: "result")
- **Extraction**: `pkg/workflow/metrics.go` → `ExtractJSONCost()`
- **Storage**: `EstimatedCost` field in `LogMetrics` struct
- **Display**: `logs_report.go` formats as currency using console rendering

### Validating Cost Data

To verify cost calculations:

```bash
# Get workflow run logs
gh aw logs --json > logs.json

# Check a specific run's cost data
jq '.runs[] | select(.database_id == 12345) | {tokens: .token_usage, cost: .estimated_cost, "cost_per_k": (.estimated_cost / .token_usage * 1000)}' logs.json
```

Note: The `cost_per_k` calculation above will give you an average, but remember it's not representative of the actual per-token pricing due to the mix of token types.

## Related Files

- Cost extraction: `pkg/workflow/metrics.go`
- Cost display: `pkg/cli/logs_report.go`
- Claude log parsing: `pkg/workflow/claude_logs.go`
- Copilot log parsing: `pkg/workflow/copilot_logs.go`
