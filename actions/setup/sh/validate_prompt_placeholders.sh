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

# Fallback: substitute __GH_AW_WIKI_NOTE__ with an empty string if still present.
# This handles workflows compiled before GH_AW_WIKI_NOTE was added to the substitution
# step for non-wiki repos. The wiki note is always optional (empty = no wiki note),
# so it is safe to clear it here rather than failing with a confusing placeholder error.
# If you see this message, run `gh aw update` to recompile your workflow and avoid the fallback.
if grep -q "__GH_AW_WIKI_NOTE__" "$PROMPT_FILE"; then
    echo "⚠️  Warning: __GH_AW_WIKI_NOTE__ was not substituted by the substitution step."
    echo "   Applying fallback: replacing with empty string."
    echo "   To resolve this permanently, run 'gh aw update' to recompile your workflow."
    sed 's/__GH_AW_WIKI_NOTE__//g' "$PROMPT_FILE" > "$PROMPT_FILE.tmp" && mv "$PROMPT_FILE.tmp" "$PROMPT_FILE"
fi

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
