#!/bin/bash
# Validate that all expression placeholders have been properly substituted
# This script checks that the prompt file doesn't contain any unreplaced placeholders

set -e

PROMPT_FILE="${GH_AW_PROMPT:-/tmp/gh-aw/aw-prompts/prompt.txt}"

if [ ! -f "$PROMPT_FILE" ]; then
    echo "❌ Error: Prompt file not found at $PROMPT_FILE"
    exit 1
fi

echo "🔍 Validating prompt placeholders..."

# Check for unreplaced environment variable placeholders (format: __GH_AW_*__)
if grep -q "__GH_AW_" "$PROMPT_FILE"; then
    echo "❌ Error: Found unreplaced placeholders in prompt file:"
    echo ""
    grep -n "__GH_AW_" "$PROMPT_FILE" | head -20
    echo ""
    echo "These placeholders should have been replaced with their actual values."
    echo "This indicates a problem with the placeholder substitution step."
    exit 1
fi

# Check for unreplaced GitHub expression syntax (format: ${{ ... }})
# Note: We allow ${{ }} in certain contexts like handlebars templates, 
# but not in the actual prompt content that should have been substituted
if grep -q '\${{[^}]*}}' "$PROMPT_FILE"; then
    # Count occurrences
    COUNT=$(grep -o '\${{[^}]*}}' "$PROMPT_FILE" | wc -l)
    
    # Show a sample of the problematic expressions
    echo "⚠️  Warning: Found $COUNT potential unreplaced GitHub expressions in prompt:"
    echo ""
    grep -n '\${{[^}]*}}' "$PROMPT_FILE" | head -10
    echo ""
    echo "Note: Some expressions may be intentional (e.g., in handlebars templates)."
    echo "Please verify these are expected."
fi

# Count total lines and characters for informational purposes
LINE_COUNT=$(wc -l < "$PROMPT_FILE")
CHAR_COUNT=$(wc -c < "$PROMPT_FILE")
WORD_COUNT=$(wc -w < "$PROMPT_FILE")

echo "✅ Placeholder validation complete"
echo "📊 Prompt statistics:"
echo "   - Lines: $LINE_COUNT"
echo "   - Characters: $CHAR_COUNT"
echo "   - Words: $WORD_COUNT"
