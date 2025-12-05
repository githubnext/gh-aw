#!/usr/bin/env python3
"""
Example Python handler for safe-inputs
Calculates the sum of two numbers
"""
import json
import sys

# Read inputs from stdin
try:
    inputs = json.loads(sys.stdin.read()) if not sys.stdin.isatty() else {}
except (json.JSONDecodeError, Exception):
    inputs = {}

a = inputs.get('a', 0)
b = inputs.get('b', 0)

# Calculate and return result as JSON
result = {
    "sum": a + b,
    "calculation": f"{a} + {b} = {a + b}"
}

print(json.dumps(result))
