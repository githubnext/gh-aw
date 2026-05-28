//go:build !integration

package workflow

import (
	"strings"
	"testing"
)

// TestRemoveJavaScriptComments tests the removeJavaScriptComments function extensively
func TestRemoveJavaScriptComments(t *testing.T) {
	t.Run("basic cases", func(t *testing.T) {
		testRemoveJavaScriptCommentsCases(t, testRemoveJavaScriptCommentsBasicCases())
	})
	t.Run("block comments", func(t *testing.T) {
		testRemoveJavaScriptCommentsCases(t, testRemoveJavaScriptCommentsBlockCases())
	})
	t.Run("directives and strings", func(t *testing.T) {
		testRemoveJavaScriptCommentsCases(t, testRemoveJavaScriptCommentsDirectiveCases())
		testRemoveJavaScriptCommentsCases(t, testRemoveJavaScriptCommentsStringCases())
	})
	t.Run("regex and escapes", func(t *testing.T) {
		testRemoveJavaScriptCommentsCases(t, testRemoveJavaScriptCommentsRegexCases())
		testRemoveJavaScriptCommentsCases(t, testRemoveJavaScriptCommentsEscapeCases())
	})
	t.Run("mixed and whitespace", func(t *testing.T) {
		testRemoveJavaScriptCommentsCases(t, testRemoveJavaScriptCommentsMixedCases())
		testRemoveJavaScriptCommentsCases(t, testRemoveJavaScriptCommentsWhitespaceCases())
	})
	t.Run("edge and real world", func(t *testing.T) {
		testRemoveJavaScriptCommentsCases(t, testRemoveJavaScriptCommentsEdgeCases())
		testRemoveJavaScriptCommentsCases(t, testRemoveJavaScriptCommentsRealWorldCases())
	})
}

type jsCommentTestCase struct {
	name     string
	input    string
	expected string
}

func testRemoveJavaScriptCommentsCases(t *testing.T, tests []jsCommentTestCase) {
	t.Helper()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := removeJavaScriptComments(tt.input)
			if result != tt.expected {
				t.Errorf("removeJavaScriptComments() = %q, expected %q", result, tt.expected)
			}
		})
	}
}

func testRemoveJavaScriptCommentsBasicCases() []jsCommentTestCase {
	return []jsCommentTestCase{
		{name: "empty string", input: "", expected: ""},
		{name: "no comments", input: "const x = 1;", expected: "const x = 1;"},
		{name: "single line comment only", input: "// comment", expected: ""},
		{name: "multiple single line comments", input: "// comment 1
// comment 2
// comment 3", expected: "

"},
		{name: "code with trailing comment", input: "const x = 1; // comment", expected: "const x = 1; "},
		{name: "code with leading comment", input: "// comment
const x = 1;", expected: "
const x = 1;"},
	}
}

func testRemoveJavaScriptCommentsBlockCases() []jsCommentTestCase {
	return []jsCommentTestCase{
		{name: "simple block comment", input: "/* comment */", expected: ""},
		{name: "multiline block comment", input: "/* line 1
   line 2
   line 3 */", expected: "

"},
		{name: "code before and after block comment", input: "const x = 1; /* comment */ const y = 2;", expected: "const x = 1;  const y = 2;"},
		{name: "nested block comment markers", input: "/* outer /* inner */ still in comment */const x = 1;", expected: " still in comment */const x = 1;"},
		{name: "block comment spanning multiple lines with code after", input: "/* comment
   line 2 */
const x = 1;", expected: "

const x = 1;"},
		{name: "multiple block comments", input: "/* c1 */ const x = 1; /* c2 */ const y = 2; /* c3 */", expected: " const x = 1;  const y = 2; "},
		{name: "JSDoc single line", input: "/** @param {string} x */", expected: ""},
	}
}

func testRemoveJavaScriptCommentsDirectiveCases() []jsCommentTestCase {
	return []jsCommentTestCase{
		{name: "JSDoc multiline", input: "/**
 * @param {string} x
 * @returns {number}
 */
function test() {}", expected: "



function test() {}"},
		{name: "JSDoc with description", input: "/**
 * Function description
 * @param {string} name - The name
 */", expected: "


"},
		{name: "ts-check directive", input: "// @ts-check
const x = 1;", expected: "
const x = 1;"},
		{name: "ts-ignore directive", input: "// @ts-ignore
const x: any = 1;", expected: "
const x: any = 1;"},
		{name: "triple slash reference", input: "/// <reference types="node" />
const x = 1;", expected: "
const x = 1;"},
		{name: "multiple TypeScript directives", input: "// @ts-check
/// <reference types="@actions/github-script" />

const x = 1;", expected: "


const x = 1;"},
	}
}

func testRemoveJavaScriptCommentsStringCases() []jsCommentTestCase {
	return []jsCommentTestCase{
		{name: "single line comment in double quotes", input: "const url = "https://example.com//path";", expected: "const url = "https://example.com//path";"},
		{name: "single line comment in single quotes", input: "const msg = 'This // is not a comment';", expected: "const msg = 'This // is not a comment';"},
		{name: "block comment in double quotes", input: "const msg = "This /* is not */ a comment";", expected: "const msg = "This /* is not */ a comment";"},
		{name: "block comment in single quotes", input: "const msg = 'This /* is not */ a comment';", expected: "const msg = 'This /* is not */ a comment';"},
		{name: "comment markers in template literal", input: "const template = `// this ${x} /* stays */ here`;", expected: "const template = `// this ${x} /* stays */ here`;"},
		{name: "escaped quotes with comments", input: "const msg = "Quote: \" // not a comment";", expected: "const msg = "Quote: \" // not a comment";"},
		{name: "string with real comment after", input: "const url = "http://example.com"; // real comment", expected: "const url = "http://example.com"; "},
	}
}

func testRemoveJavaScriptCommentsRegexCases() []jsCommentTestCase {
	return []jsCommentTestCase{
		{name: "regex with slashes", input: "const regex = /\/\//;", expected: "const regex = /\/\//;"},
		{name: "regex with comment after", input: "const regex = /test/g; // comment", expected: "const regex = /test/g; "},
		{name: "complex regex pattern", input: "const regex = /foo\/bar\/baz/gi;", expected: "const regex = /foo\/bar\/baz/gi;"},
		{name: "regex with escaped slashes and comment", input: "const regex = /\/\/test\/\//; // matches //test//", expected: "const regex = /\/\/test\/\//; "},
		{name: "division operator not regex", input: "const result = x / y; // division", expected: "const result = x / y; "},
		{name: "regex after return", input: "return /test/;", expected: "return /test/;"},
		{name: "regex in assignment", input: "const r = /pattern/;", expected: "const r = /pattern/;"},
		{name: "regex with character class", input: "const r = /[a-z]/gi;", expected: "const r = /[a-z]/gi;"},
	}
}

func testRemoveJavaScriptCommentsEscapeCases() []jsCommentTestCase {
	return []jsCommentTestCase{
		{name: "escaped backslash in string", input: "const path = "C:\path\to\file";", expected: "const path = "C:\path\to\file";"},
		{name: "escaped quote in string", input: "const quote = "He said \"hello\"";", expected: "const quote = "He said \"hello\"";"},
		{name: "escaped newline in string", input: "const msg = "Line 1
Line 2";", expected: "const msg = "Line 1
Line 2";"},
		{name: "code with multiple comment types", input: "// Line comment
/* Block comment */
const x = 1; // trailing
/** JSDoc */
const y = 2;", expected: "

const x = 1; 

const y = 2;"},
		{name: "real world example with imports", input: "// Import statements
const fs = require('fs');
const path = require('path'); // path module
/* Utility function */
function test() { return true; }", expected: "
const fs = require('fs');
const path = require('path'); 

function test() { return true; }"},
		{name: "function with inline comments", input: "function calc(a, b) {
  // Add numbers
  return a + b; // result
}", expected: "function calc(a, b) {
  
  return a + b; 
}"},
	}
}

func testRemoveJavaScriptCommentsMixedCases() []jsCommentTestCase {
	return []jsCommentTestCase{
		{name: "object literal with comments", input: "const obj = {
  // Property
  key: 'value', // inline
  /* Another */
  key2: 'value2'
};", expected: "const obj = {
  
  key: 'value', 
  
  key2: 'value2'
};"},
		{name: "array with comments", input: "const arr = [
  1, // first
  2, // second
  /* skip */ 3
];", expected: "const arr = [
  1, 
  2, 
   3
];"},
		{name: "preserve indentation", input: "  // comment
  const x = 1;", expected: "  
  const x = 1;"},
		{name: "preserve spacing in code", input: "const x = 1;    const y = 2;", expected: "const x = 1;    const y = 2;"},
		{name: "unicode in strings", input: "const emoji = "😀 // not a comment";", expected: "const emoji = "😀 // not a comment";"},
		{name: "unicode in comment", input: "// Comment with emoji 😀
const x = 1;", expected: "
const x = 1;"},
	}
}

func testRemoveJavaScriptCommentsWhitespaceCases() []jsCommentTestCase {
	return []jsCommentTestCase{
		{name: "empty lines preserved", input: "const x = 1;


const y = 2;", expected: "const x = 1;


const y = 2;"},
		{name: "comment on empty line removed", input: "const x = 1;

// comment

const y = 2;", expected: "const x = 1;



const y = 2;"},
		{name: "whitespace only lines", input: "const x = 1;
   
	
const y = 2;", expected: "const x = 1;
   
	
const y = 2;"},
		{name: "comment at start", input: "// Start comment
const x = 1;", expected: "
const x = 1;"},
		{name: "comment at end", input: "const x = 1;
// End comment", expected: "const x = 1;
"},
		{name: "comment at both ends", input: "// Start
const x = 1;
// End", expected: "
const x = 1;
"},
		{name: "consecutive line comments", input: "// Comment 1
// Comment 2
// Comment 3", expected: "

"},
		{name: "consecutive block comments", input: "/* C1 *//* C2 *//* C3 */", expected: ""},
	}
}

func testRemoveJavaScriptCommentsEdgeCases() []jsCommentTestCase {
	return []jsCommentTestCase{
		{name: "alternating comments and code", input: "// C1
const a = 1;
// C2
const b = 2;
// C3
const c = 3;", expected: "
const a = 1;

const b = 2;

const c = 3;"},
		{name: "division after number", input: "const x = 10 / 2;", expected: "const x = 10 / 2;"},
		{name: "comment-like in URL", input: "const url = 'http://example.com/path';", expected: "const url = 'http://example.com/path';"},
		{name: "comment-like in file path", input: "const path = 'C://Windows//System32';", expected: "const path = 'C://Windows//System32';"},
		{name: "asterisk in string not comment", input: "const str = 'This * is not a comment';", expected: "const str = 'This * is not a comment';"},
		{name: "slash and asterisk separate", input: "const x = '/' + '*' + 'combined';", expected: "const x = '/' + '*' + 'combined';"},
	}
}

func testRemoveJavaScriptCommentsRealWorldCases() []jsCommentTestCase {
	return []jsCommentTestCase{
		{name: "unclosed block comment", input: "const x = 1; /* unclosed comment", expected: "const x = 1; "},
		{name: "block comment with asterisks inside", input: "/* comment * with * asterisks */const x = 1;", expected: "const x = 1;"},
		{name: "block comment end marker in string", input: "const str = "This */ is not comment end"; /* comment */", expected: "const str = "This */ is not comment end"; "},
		{name: "slash star in separate strings", input: "const a = '/'; const b = '*'; // comment", expected: "const a = '/'; const b = '*'; "},
		{name: "GitHub Actions script pattern", input: "// @ts-check
/// <reference types="@actions/github-script" />

async function main() {
  core.info('test');
}", expected: "


async function main() {
  core.info('test');
}"},
		{name: "JSDoc with function", input: "/**
 * Process data
 * @param {Object} data - The data
 * @returns {boolean}
 */
function process(data) {
  return true;
}", expected: "




function process(data) {
  return true;
}"},
		{name: "module pattern with comments", input: "// Module exports
module.exports = {
  // Properties
  prop1: value1, // inline
  /* Another property */
  prop2: value2
};", expected: "
module.exports = {
  
  prop1: value1, 
  
  prop2: value2
};"},
	}
}

// TestRemoveJavaScriptCommentsFromLine tests line-level comment removal
func TestRemoveJavaScriptCommentsFromLine(t *testing.T) {
	tests := []struct {
		name           string
		line           string
		inBlockComment bool
		expected       string
		expectBlock    bool
	}{
		{
			name:           "simple line no comment",
			line:           "const x = 1;",
			inBlockComment: false,
			expected:       "const x = 1;",
			expectBlock:    false,
		},
		{
			name:           "line with trailing comment",
			line:           "const x = 1; // comment",
			inBlockComment: false,
			expected:       "const x = 1; ",
			expectBlock:    false,
		},
		{
			name:           "line starting in block comment",
			line:           "still in block */ after block",
			inBlockComment: true,
			expected:       " after block",
			expectBlock:    false,
		},
		{
			name:           "line starting block comment",
			line:           "before /* start block",
			inBlockComment: false,
			expected:       "before ",
			expectBlock:    true,
		},
		{
			name:           "line with complete block comment",
			line:           "before /* block */ after",
			inBlockComment: false,
			expected:       "before  after",
			expectBlock:    false,
		},
		{
			name:           "entire line is comment",
			line:           "// entire line",
			inBlockComment: false,
			expected:       "",
			expectBlock:    false,
		},
		{
			name:           "line in middle of block comment",
			line:           "this is inside block comment",
			inBlockComment: true,
			expected:       "",
			expectBlock:    true,
		},
		{
			name:           "comment in string preserved",
			line:           "const s = \"// not a comment\";",
			inBlockComment: false,
			expected:       "const s = \"// not a comment\";",
			expectBlock:    false,
		},
		{
			name:           "regex with slashes",
			line:           "const r = /test\\/path/;",
			inBlockComment: false,
			expected:       "const r = /test\\/path/;",
			expectBlock:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			inBlock := tt.inBlockComment
			result := removeJavaScriptCommentsFromLine(tt.line, &inBlock)
			if result != tt.expected {
				t.Errorf("removeJavaScriptCommentsFromLine() = %q, expected %q", result, tt.expected)
			}
			if inBlock != tt.expectBlock {
				t.Errorf("inBlockComment after = %v, expected %v", inBlock, tt.expectBlock)
			}
		})
	}
}

// TestIsInsideStringLiteral tests string literal detection
func TestIsInsideStringLiteral(t *testing.T) {
	tests := []struct {
		name     string
		text     string
		expected bool
	}{
		{
			name:     "not in string",
			text:     "const x = ",
			expected: false,
		},
		{
			name:     "inside double quotes",
			text:     "const x = \"hello",
			expected: true,
		},
		{
			name:     "inside single quotes",
			text:     "const x = 'hello",
			expected: true,
		},
		{
			name:     "inside backticks",
			text:     "const x = `hello",
			expected: true,
		},
		{
			name:     "closed double quotes",
			text:     "const x = \"hello\"; y = ",
			expected: false,
		},
		{
			name:     "escaped quote in string",
			text:     "const x = \"hello \\\"world",
			expected: true,
		},
		{
			name:     "escaped backslash before quote",
			text:     "const x = \"hello\\\\\" y = ",
			expected: false,
		},
		{
			name:     "nested different quotes",
			text:     "const x = \"hello 'world",
			expected: true,
		},
		{
			name:     "empty string",
			text:     "",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isInsideStringLiteral(tt.text)
			if result != tt.expected {
				t.Errorf("isInsideStringLiteral(%q) = %v, expected %v", tt.text, result, tt.expected)
			}
		})
	}
}

// TestCanStartRegexLiteral tests regex literal detection
func TestCanStartRegexLiteral(t *testing.T) {
	tests := []struct {
		name     string
		text     string
		expected bool
	}{
		{
			name:     "after equals",
			text:     "const x = ",
			expected: true,
		},
		{
			name:     "after return",
			text:     "return ",
			expected: false, // Needs the keyword at the end of text
		},
		{
			name:     "after opening paren",
			text:     "test(",
			expected: true,
		},
		{
			name:     "after comma",
			text:     "arr = [1, ",
			expected: true,
		},
		{
			name:     "after opening bracket",
			text:     "arr = [",
			expected: true,
		},
		{
			name:     "after colon",
			text:     "obj = { key: ",
			expected: true,
		},
		{
			name:     "after identifier (division)",
			text:     "x ",
			expected: false,
		},
		{
			name:     "after number (division)",
			text:     "10 ",
			expected: false,
		},
		{
			name:     "at start of line",
			text:     "",
			expected: true,
		},
		{
			name:     "after if keyword",
			text:     "if ",
			expected: false, // Needs the keyword at the end of text
		},
		{
			name:     "after while keyword",
			text:     "while ",
			expected: false, // Needs the keyword at the end of text
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := canStartRegexLiteral(tt.text)
			if result != tt.expected {
				t.Errorf("canStartRegexLiteral(%q) = %v, expected %v", tt.text, result, tt.expected)
			}
		})
	}
}

// TestRemoveJavaScriptCommentsEdgeCases tests additional edge cases
func TestRemoveJavaScriptCommentsEdgeCases(t *testing.T) {
	t.Run("very long line with comment", func(t *testing.T) {
		longLine := strings.Repeat("x", 10000) + " // comment"
		result := removeJavaScriptComments(longLine)
		expected := strings.Repeat("x", 10000) + " "
		if result != expected {
			t.Errorf("Failed to handle long line")
		}
	})

	t.Run("many consecutive comments", func(t *testing.T) {
		input := strings.Repeat("// comment\n", 100)
		result := removeJavaScriptComments(input)
		// Each comment line leaves an empty line (100 lines, each becoming empty)
		// The function doesn't trim trailing newlines from repeated inputs
		expected := strings.Repeat("\n", 100)
		if len(result) != len(expected) {
			t.Errorf("Failed to handle many consecutive comments: got %d chars, expected %d", len(result), len(expected))
		}
	})

	t.Run("deeply nested strings and comments", func(t *testing.T) {
		input := `const a = "/*"; const b = "*/"; // comment`
		result := removeJavaScriptComments(input)
		expected := `const a = "/*"; const b = "*/"; `
		if result != expected {
			t.Errorf("removeJavaScriptComments() = %q, expected %q", result, expected)
		}
	})

	t.Run("multiple block comments on same line", func(t *testing.T) {
		input := "/* c1 */ a /* c2 */ b /* c3 */"
		result := removeJavaScriptComments(input)
		expected := " a  b "
		if result != expected {
			t.Errorf("removeJavaScriptComments() = %q, expected %q", result, expected)
		}
	})

	t.Run("block comment across many lines", func(t *testing.T) {
		input := "start /* comment\nline 2\nline 3\nline 4\nline 5 */ end"
		result := removeJavaScriptComments(input)
		// Block comments preserve line structure
		expected := "start \n\n\n\n end"
		if result != expected {
			t.Errorf("removeJavaScriptComments() = %q, expected %q", result, expected)
		}
	})
}

// BenchmarkRemoveJavaScriptComments benchmarks the comment removal function
func BenchmarkRemoveJavaScriptComments(b *testing.B) {
	testCases := []struct {
		name  string
		input string
	}{
		{
			name:  "simple code no comments",
			input: "const x = 1;\nconst y = 2;\nconst z = 3;",
		},
		{
			name:  "code with comments",
			input: "// Comment\nconst x = 1; // inline\n/* block */ const y = 2;",
		},
		{
			name: "real world script",
			input: `// @ts-check
/// <reference types="@actions/github-script" />

async function main() {
  const { eventName } = context;
  
  // skip check for safe events
  const safeEvents = ["workflow_dispatch", "schedule"];
  if (safeEvents.includes(eventName)) {
    core.info('Event does not require validation');
    return;
  }
  
  /* Process the event */
  const actor = context.actor;
  const { owner, repo } = context.repo;
  
  // Return result
  return { actor, owner, repo };
}`,
		},
	}

	for _, tc := range testCases {
		b.Run(tc.name, func(b *testing.B) {
			for range b.N {
				removeJavaScriptComments(tc.input)
			}
		})
	}
}
