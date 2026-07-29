---
# Default AI-credits pricing fallback.
# Use in workflows that may invoke models without catalog pricing, to prevent
# missing_model_pricing / unknown_model_ai_credits failures.
# Rates: $5 per M input tokens, $25 per M output tokens (conservative GPT-4-class defaults).
#
# Usage:
#   imports:
#     - shared/default-ai-credits-pricing.md
models:
  default-ai-credits-pricing:
    input: 5.0
    output: 25.0
---
