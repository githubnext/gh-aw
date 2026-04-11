#!/bin/bash
# Validate that all expression placeholders have been properly substituted
# This script checks that the prompt file doesn't contain any unreplaced placeholders

set -e

PROMPT_FILE="${GH_AW_PROMPT:-/tmp/gh-aw/aw-prompts/prompt.txt}"
MANIFEST_FILE="${PROMPT_FILE}.placeholders"

if [ ! -f "$PROMPT_FILE" ]; then
    echo "❌ Error: Prompt file not found at $PROMPT_FILE"
    exit 1
fi

echo "🔍 Validating prompt placeholders..."

# Check for unreplaced environment variable placeholders (format: __GH_AW_*__)
# If a manifest file exists (written by substitute_placeholders.cjs), check only
# for the specific placeholders that were supposed to be substituted. This avoids
# false positives when substituted values contain __GH_AW_*__ patterns (e.g., issue
# titles that mention placeholder names).
if [ -f "$MANIFEST_FILE" ]; then
    FOUND_PLACEHOLDERS=""
    while IFS= read -r placeholder; do
        # Skip empty lines
        [ -z "$placeholder" ] && continue
        if grep -qF "$placeholder" "$PROMPT_FILE"; then
            if [ -z "$FOUND_PLACEHOLDERS" ]; then
                FOUND_PLACEHOLDERS="$placeholder"
            else
                FOUND_PLACEHOLDERS="$FOUND_PLACEHOLDERS
$placeholder"
            fi
        fi
    done < "$MANIFEST_FILE"

    if [ -n "$FOUND_PLACEHOLDERS" ]; then
        echo "❌ Error: Found unreplaced placeholders in prompt file:"
        echo ""
        echo "$FOUND_PLACEHOLDERS" | while IFS= read -r p; do
            grep -nF "$p" "$PROMPT_FILE" | head -5
        done
        echo ""
        echo "These placeholders should have been replaced with their actual values."
        echo "This indicates a problem with the placeholder substitution step."
        exit 1
    fi
else
    # Fallback: no manifest file, use broad pattern matching (legacy behavior)
    if grep -q "__GH_AW_" "$PROMPT_FILE"; then
        echo "❌ Error: Found unreplaced placeholders in prompt file:"
        echo ""
        grep -n "__GH_AW_" "$PROMPT_FILE" | head -20
        echo ""
        echo "These placeholders should have been replaced with their actual values."
        echo "This indicates a problem with the placeholder substitution step."
        exit 1
    fi
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
