#!/usr/bin/env python3
"""
Test script for drain3_server.py
Validates the server code without requiring dependencies to be installed.
"""
import ast
import sys
from pathlib import Path


def test_syntax():
    """Test that the Python file has valid syntax."""
    script_path = Path(__file__).parent / "drain3_server.py"
    with open(script_path, "r") as f:
        code = f.read()
    
    try:
        ast.parse(code)
        print("✓ Python syntax is valid")
        return True
    except SyntaxError as e:
        print(f"✗ Syntax error: {e}")
        return False


def test_structure():
    """Test that the file has expected structure."""
    script_path = Path(__file__).parent / "drain3_server.py"
    with open(script_path, "r") as f:
        code = f.read()
    
    tree = ast.parse(code)
    
    # Check for required imports
    imports = []
    for node in ast.walk(tree):
        if isinstance(node, ast.ImportFrom):
            imports.append(node.module)
    
    required_imports = ["fastmcp"]
    missing = [imp for imp in required_imports if not any(imp in i for i in imports if i)]
    
    if missing:
        print(f"✗ Missing imports: {missing}")
        return False
    print("✓ Required imports present")
    
    # Check for tool decorators
    tool_functions = []
    for node in ast.walk(tree):
        if isinstance(node, ast.FunctionDef):
            for decorator in node.decorator_list:
                # Check for @mcp.tool() (Call node)
                if isinstance(decorator, ast.Call):
                    if isinstance(decorator.func, ast.Attribute) and decorator.func.attr == "tool":
                        tool_functions.append(node.name)
    
    expected_tools = ["index_file", "query_file", "list_templates", "list_clusters", "cluster_stats", "find_anomalies", "compare_runs", "search_pattern"]
    missing_tools = [tool for tool in expected_tools if tool not in tool_functions]
    
    if missing_tools:
        print(f"✗ Missing tool functions: {missing_tools}")
        return False
    print(f"✓ All expected tool functions found: {', '.join(expected_tools)}")
    
    # Check for main block
    has_main = False
    for node in ast.walk(tree):
        if isinstance(node, ast.If):
            if isinstance(node.test, ast.Compare):
                if any(isinstance(comp, ast.Eq) for comp in node.test.ops):
                    has_main = True
                    break
    
    if not has_main:
        print("✗ No main block found")
        return False
    print("✓ Main block present")
    
    return True


def test_no_invalid_params():
    """Test that mcp.run() doesn't have invalid parameters and decorators are called correctly."""
    script_path = Path(__file__).parent / "drain3_server.py"
    with open(script_path, "r") as f:
        code = f.read()
    
    tree = ast.parse(code)
    
    # Check decorators are called (with parentheses)
    for node in ast.walk(tree):
        if isinstance(node, ast.FunctionDef):
            for decorator in node.decorator_list:
                if isinstance(decorator, ast.Attribute) and decorator.attr == "tool":
                    # This should be a Call node (mcp.tool()), not just an Attribute (mcp.tool)
                    print(f"✗ @mcp.tool decorator on '{node.name}' should be called with parentheses: @mcp.tool()")
                    return False
    
    # Find mcp.run() calls
    for node in ast.walk(tree):
        if isinstance(node, ast.Call):
            if (isinstance(node.func, ast.Attribute) and 
                node.func.attr == "run"):
                # Check keywords
                keywords = [kw.arg for kw in node.keywords]
                # path parameter should not be in mcp.run() for streamable-http
                if "path" in keywords:
                    print("✗ mcp.run() should not have 'path' parameter for streamable-http transport")
                    return False
                print(f"✓ mcp.run() parameters look correct: {', '.join(keywords)}")
                
    print("✓ All decorators are called correctly with parentheses")
    return True


def main():
    """Run all tests."""
    print("Testing drain3_server.py...")
    print()
    
    tests = [
        ("Syntax validation", test_syntax),
        ("Structure validation", test_structure),
        ("Parameter validation", test_no_invalid_params),
    ]
    
    results = []
    for name, test_func in tests:
        print(f"Running: {name}")
        result = test_func()
        results.append(result)
        print()
    
    if all(results):
        print("✓ All tests passed!")
        return 0
    else:
        print("✗ Some tests failed")
        return 1


if __name__ == "__main__":
    sys.exit(main())
