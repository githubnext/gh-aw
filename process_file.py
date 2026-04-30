#!/usr/bin/env python3
"""
Script to migrate fmt.Fprintf/Fprintln(os.Stderr) calls to console helpers.
This processes one file at a time and outputs the modified content.
"""

import re
import sys

def should_skip_conversion(line, prev_lines):
    """Determine if this line should be skipped."""
    # Skip blank lines
    if 'fmt.Fprintln(os.Stderr, "")' in line or line.strip() == 'fmt.Fprintln(os.Stderr)':
        return True
    if 'fmt.Fprintf(os.Stderr, "\\n")' in line and 'fmt.Fprintf(os.Stderr, "\\n")' == line.strip():
        return True
    # Skip complex formatting with padding/alignment
    if re.search(r'%-?\d+[sd]', line):
        return True
    # Skip if already using console.Format
    if 'console.Format' in line:
        return True
    return False

def determine_helper(message_content):
    """Determine the appropriate console helper based on message content."""
    msg_lower = message_content.lower()
    
    # Error patterns
    if any(p in msg_lower for p in ['error:', 'failed', 'failure', 'cannot', 'unable', 'err.error()']):
        return 'FormatErrorMessage'
    
    # Success patterns
    if any(p in msg_lower for p in ['success', 'done!', 'complete', ' enabled', ' disabled', 'created', 'updated', 'removed']):
        return 'FormatSuccessMessage'
    
    # Progress patterns (present continuous)
    if any(p in msg_lower for p in ['downloading', 'processing', 'compiling', 'running', 'checking', 'fetching', 'loading', 'building', 'installing', 'analyzing', 'creating']):
        return 'FormatProgressMessage'
    
    # Warning patterns
    if any(p in msg_lower for p in ['warning:', 'note:', 'deprecated', 'caution']):
        return 'FormatWarningMessage'
    
    # Command patterns
    if 'gh aw ' in message_content or message_content.strip().startswith('`'):
        return 'FormatCommandMessage'
    
    # List item (starts with spaces/indent)
    if message_content.lstrip().startswith('  ') and not message_content.lstrip().startswith('   '):
        return 'FormatListItem'
    
    # Default to info
    return 'FormatInfoMessage'

def clean_prefix(msg, helper):
    """Remove redundant prefixes based on the helper."""
    if helper == 'FormatErrorMessage':
        msg = re.sub(r'^Error:\s*', '', msg, flags=re.IGNORECASE)
        msg = re.sub(r'^Failed:\s*', '', msg, flags=re.IGNORECASE)
    elif helper == 'FormatWarningMessage':
        msg = re.sub(r'^Warning:\s*', '', msg, flags=re.IGNORECASE)
        msg = re.sub(r'^Note:\s*', '', msg, flags=re.IGNORECASE)
    elif helper == 'FormatSuccessMessage':
        msg = re.sub(r'^Success:\s*', '', msg, flags=re.IGNORECASE)
        msg = re.sub(r'^✓\s*', '', msg)
    return msg

def convert_fprintln(line):
    """Convert fmt.Fprintln(os.Stderr, msg) to use console helper."""
    # Pattern: fmt.Fprintln(os.Stderr, "literal message")
    match = re.search(r'fmt\.Fprintln\(os\.Stderr,\s*"([^"]+)"\)', line)
    if match:
        msg = match.group(1)
        helper = determine_helper(msg)
        msg = clean_prefix(msg, helper)
        # Remove \n if present at the end
        msg = msg.rstrip('\\n')
        replacement = f'fmt.Fprintln(os.Stderr, console.{helper}("{msg}"))'
        return line.replace(match.group(0), replacement)
    
    # Pattern: fmt.Fprintln(os.Stderr, console.FormatXXX(...))
    # Already converted, skip
    if 'console.Format' in line:
        return line
    
    return line

def convert_fprintf(line):
    """Convert fmt.Fprintf(os.Stderr, format, args...) to use console helper."""
    # Simple pattern: fmt.Fprintf(os.Stderr, "message\n")
    match = re.search(r'fmt\.Fprintf\(os\.Stderr,\s*"([^"]+)\\n"\)', line)
    if match:
        msg = match.group(1)
        # Skip if it has format verbs
        if '%' in msg:
            return line
        helper = determine_helper(msg)
        msg = clean_prefix(msg, helper)
        replacement = f'fmt.Fprintln(os.Stderr, console.{helper}("{msg}"))'
        return line.replace(match.group(0), replacement)
    
    return line

def process_file(content):
    """Process file content and return modified version."""
    lines = content.split('\n')
    modified_lines = []
    needs_console_import = False
    has_console_import = False
    
    # Check if console is already imported
    in_import_block = False
    for line in lines:
        if 'import (' in line:
            in_import_block = True
        elif in_import_block and ')' in line:
            in_import_block = False
        
        if 'github.com/github/gh-aw/pkg/console' in line:
            has_console_import = True
            break
    
    for i, line in enumerate(lines):
        prev_lines = lines[max(0, i-3):i]
        
        # Skip if shouldn't convert
        if should_skip_conversion(line, prev_lines):
            modified_lines.append(line)
            continue
        
        # Try conversions
        original = line
        if 'fmt.Fprintln(os.Stderr' in line:
            line = convert_fprintln(line)
        elif 'fmt.Fprintf(os.Stderr' in line:
            line = convert_fprintf(line)
        
        if line != original:
            needs_console_import = True
        
        modified_lines.append(line)
    
    # Add console import if needed and not present
    if needs_console_import and not has_console_import:
        # Find import block and add console
        for i, line in enumerate(modified_lines):
            if 'import (' in line:
                # Find a good place to insert (after other pkg imports)
                j = i + 1
                while j < len(modified_lines) and modified_lines[j].strip() and ')' not in modified_lines[j]:
                    j += 1
                # Insert before the closing paren or at the end of imports
                insert_pos = j
                for k in range(i+1, j):
                    if 'github.com/github/gh-aw/pkg/' in modified_lines[k]:
                        insert_pos = k + 1
                modified_lines.insert(insert_pos, '\t"github.com/github/gh-aw/pkg/console"')
                break
    
    return '\n'.join(modified_lines)

if __name__ == '__main__':
    if len(sys.argv) != 2:
        print("Usage: process_file.py <filepath>", file=sys.stderr)
        sys.exit(1)
    
    filepath = sys.argv[1]
    with open(filepath, 'r') as f:
        content = f.read()
    
    modified = process_file(content)
    
    with open(filepath, 'w') as f:
        f.write(modified)
    
    print(f"Processed: {filepath}")
