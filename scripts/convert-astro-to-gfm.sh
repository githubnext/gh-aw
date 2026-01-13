#!/bin/bash
# Convert Astro adornments to GitHub Flavored Markdown alerts
# Usage: ./scripts/convert-astro-to-gfm.sh <file>

set -e

if [ $# -ne 1 ]; then
    echo "Usage: $0 <file>"
    exit 1
fi

FILE="$1"

if [ ! -f "$FILE" ]; then
    echo "Error: File '$FILE' not found"
    exit 1
fi

# Create a temporary file
TMP_FILE=$(mktemp)

# Process the file using awk
awk '
BEGIN {
    in_adornment = 0
    adornment_type = ""
    adornment_title = ""
}

# Match opening adornment with title: :::type[title]
/^:::([a-z]+)\[.*\]/ {
    match($0, /^:::([a-z]+)\[(.*)\]/, arr)
    adornment_type = toupper(arr[1])
    adornment_title = arr[2]
    
    # Special case: tip -> TIP, note -> NOTE, caution -> CAUTION, warning -> WARNING
    print "> [!" adornment_type "]"
    if (adornment_title != "") {
        print "> " adornment_title
    }
    in_adornment = 1
    next
}

# Match opening adornment without title: :::type
/^:::([a-z]+)$/ {
    match($0, /^:::([a-z]+)/, arr)
    adornment_type = toupper(arr[1])
    
    print "> [!" adornment_type "]"
    in_adornment = 1
    next
}

# Match closing adornment: :::
/^:::$/ {
    in_adornment = 0
    adornment_type = ""
    adornment_title = ""
    next
}

# Process lines inside adornment
in_adornment == 1 {
    if ($0 == "") {
        print ">"
    } else {
        print "> " $0
    }
    next
}

# Pass through all other lines unchanged
{
    print
}
' "$FILE" > "$TMP_FILE"

# Replace the original file with the converted content
mv "$TMP_FILE" "$FILE"

echo "Converted: $FILE"
