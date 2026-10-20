// @ts-check

import { describe, it, expect } from "vitest";
import { createRequire } from "module";

const require = createRequire(import.meta.url);
const { splitOnPipelineOperators, extractCommandNamesFromPipeline } = require("./bash_command_parser.cjs");
const { buildCopilotSDKPermissionHandler } = require("./copilot_sdk_permissions.cjs");

const checkPermission = buildCopilotSDKPermissionHandler({ allowedTools: ["shell(cat)", "shell(echo)", "shell(jq)", "shell(wc)"] }, () => ({ kind: "approve-once" }));

const lineEndings = [
  { name: "LF", value: "\n" },
  { name: "CRLF", value: "\r\n" },
  { name: "CR", value: "\r" },
];

const commentBodies = [
  { name: "ordinary text", text: "Read the precomputed attribution data" },
  { name: "operators", text: "ignored_tool && another_tool || third_tool | fourth_tool; fifth_tool" },
  { name: "unclosed quotes", text: `Unclosed single ' and double " quotes` },
  { name: "unclosed substitutions", text: "Ignore $(unclosed and ${unclosed and (( parentheses" },
  { name: "hashes and redirections", text: "### ignored_tool >output 2>&1 <input &" },
  { name: "trailing backslash", text: "This comment does not continue \\" },
];

describe("commented Bash script fixtures", () => {
  it.each([
    {
      name: "shebang and file header",
      script: "#!/usr/bin/env bash\n# Read the data\ncat data.json",
      segments: ["cat data.json"],
      names: ["cat"],
    },
    {
      name: "inline comment at EOF",
      script: "cat data.json # Read only",
      segments: ["cat data.json"],
      names: ["cat"],
    },
    {
      name: "comment-only final line",
      script: "cat data.json\n# Finished",
      segments: ["cat data.json"],
      names: ["cat"],
    },
    {
      name: "indented comments and blank lines",
      script: "\n  # Header\n\n\t# Another comment\n\tcat data.json\n\n  # Footer\n",
      segments: ["cat data.json"],
      names: ["cat"],
    },
    {
      name: "empty inline comments",
      script: "echo ready #\ncat data.json\t#\n#",
      segments: ["echo ready", "cat data.json"],
      names: ["echo", "cat"],
    },
    {
      name: "comment immediately after semicolon",
      script: "echo ready;# Ignore this\ncat data.json",
      segments: ["echo ready", "cat data.json"],
      names: ["echo", "cat"],
    },
    {
      name: "comment immediately after pipe",
      script: "cat data.json |# Ignore this\nwc -l",
      segments: ["cat data.json", "wc -l"],
      names: ["cat", "wc"],
    },
    {
      name: "comment immediately after AND",
      script: "cat data.json &&# Ignore this\necho done",
      segments: ["cat data.json", "echo done"],
      names: ["cat", "echo"],
    },
    {
      name: "comment immediately after OR",
      script: "cat data.json ||# Ignore this\necho missing",
      segments: ["cat data.json", "echo missing"],
      names: ["cat", "echo"],
    },
    {
      name: "commented command is not executable",
      script: "# python3 -c 'print(1)'\ncat data.json\n# another_tool",
      segments: ["cat data.json"],
      names: ["cat"],
    },
    {
      name: "single-quoted hashes and operators",
      script: "echo '# literal && not_a_command' # Actual comment\ncat data.json",
      segments: ["echo '# literal && not_a_command'", "cat data.json"],
      names: ["echo", "cat"],
    },
    {
      name: "double-quoted hashes and operators",
      script: 'echo "# literal | not_a_command" # Actual comment\ncat data.json',
      segments: ['echo "# literal | not_a_command"', "cat data.json"],
      names: ["echo", "cat"],
    },
    {
      name: "multiline single-quoted text",
      script: "echo 'first line\n# literal second line' # Actual comment\ncat data.json",
      segments: ["echo 'first line\n# literal second line'", "cat data.json"],
      names: ["echo", "cat"],
    },
    {
      name: "multiline double-quoted text",
      script: 'echo "first line\n# literal second line" # Actual comment\ncat data.json',
      segments: ['echo "first line\n# literal second line"', "cat data.json"],
      names: ["echo", "cat"],
    },
    {
      name: "hash inside filename",
      script: "cat report#draft.json # Actual comment\nwc -l report#draft.json",
      segments: ["cat report#draft.json", "wc -l report#draft.json"],
      names: ["cat", "wc"],
    },
    {
      name: "hash inside URL",
      script: "echo https://example.invalid/page#section # Actual comment\ncat data.json",
      segments: ["echo https://example.invalid/page#section", "cat data.json"],
      names: ["echo", "cat"],
    },
    {
      name: "hash adjoining empty single quotes",
      script: "echo ''#literal # Actual comment\ncat data.json",
      segments: ["echo ''#literal", "cat data.json"],
      names: ["echo", "cat"],
    },
    {
      name: "hash adjoining empty double quotes",
      script: 'echo ""#literal # Actual comment\ncat data.json',
      segments: ['echo ""#literal', "cat data.json"],
      names: ["echo", "cat"],
    },
    {
      name: "escaped hash",
      script: String.raw`echo \#literal # Actual comment` + "\ncat data.json",
      segments: [String.raw`echo \#literal`, "cat data.json"],
      names: ["echo", "cat"],
    },
    {
      name: "escaped space before literal hash",
      script: String.raw`echo value\ #literal # Actual comment` + "\ncat data.json",
      segments: [String.raw`echo value\ #literal`, "cat data.json"],
      names: ["echo", "cat"],
    },
    {
      name: "escaped tab before literal hash",
      script: "echo value\\\t#literal # Actual comment\ncat data.json",
      segments: ["echo value\\\t#literal", "cat data.json"],
      names: ["echo", "cat"],
    },
    {
      name: "escaped quote before literal hash",
      script: String.raw`echo \"#literal # Actual comment` + "\ncat data.json",
      segments: [String.raw`echo \"#literal`, "cat data.json"],
      names: ["echo", "cat"],
    },
    {
      name: "escaped operator does not start a comment",
      script: String.raw`echo \;#literal # Actual comment` + "\ncat data.json",
      segments: [String.raw`echo \;#literal`, "cat data.json"],
      names: ["echo", "cat"],
    },
    {
      name: "parameter length and prefix removal",
      script: "echo ${#value} ${path##*/} # Actual comment\ncat data.json",
      segments: ["echo ${#value} ${path##*/}", "cat data.json"],
      names: ["echo", "cat"],
    },
    {
      name: "space and hash in parameter default",
      script: "echo ${value:- # fallback} # Actual comment\ncat data.json",
      segments: ["echo ${value:- # fallback}", "cat data.json"],
      names: ["echo", "cat"],
    },
    {
      name: "nested parameter defaults",
      script: "echo ${outer:-${inner:- # fallback}} # Actual comment\ncat data.json",
      segments: ["echo ${outer:-${inner:- # fallback}}", "cat data.json"],
      names: ["echo", "cat"],
    },
    {
      name: "quoted closing brace in parameter default",
      script: "echo ${value:-'}' # fallback} # Actual comment\ncat data.json",
      segments: ["echo ${value:-'}' # fallback}", "cat data.json"],
      names: ["echo", "cat"],
    },
    {
      name: "quoted hash inside command substitution",
      script: "echo $(printf '%s' '# literal') # Actual comment\ncat data.json",
      segments: ["echo $(printf '%s' '# literal')", "cat data.json"],
      names: ["echo", "cat"],
    },
    {
      name: "continued command before inline comment",
      script: "jq \\\n  '.[]' \\\n  data.json # Filter entries\nwc -l data.json",
      segments: ["jq '.[]' data.json", "wc -l data.json"],
      names: ["jq", "wc"],
    },
    {
      name: "standalone comment after line continuation",
      script: "echo \\\n# This ends the logical command\ncat data.json",
      segments: ["echo", "cat data.json"],
      names: ["echo", "cat"],
    },
    {
      name: "assignment with inline comment",
      script: '# Configuration\nDATA=data.json # Input path\ncat "$DATA" # Read it',
      segments: ["DATA=data.json", 'cat "$DATA"'],
      names: ["cat"],
    },
    {
      name: "for loop with comments around structural keywords",
      script: '# Iterate over inputs\nfor file in first.json second.json; do # Loop body\n  cat "$file" # Read input\ndone # End loop\necho done',
      segments: ["for file in first.json second.json", "do", 'cat "$file"', "done", "echo done"],
      names: ["cat", "echo"],
    },
    {
      name: "conditional with comments on every clause",
      script: "if # Condition follows\n  grep -q community data.json # Search\nthen # Match\n  cat data.json # Read\nelse # No match\n  echo missing # Report\nfi # End",
      segments: ["if", "grep -q community data.json", "then", "cat data.json", "else", "echo missing", "fi"],
      names: ["grep", "cat", "echo"],
    },
    {
      name: "function declaration and invocation",
      script: "function report # Define helper\n{ # Start body\n  cat data.json # Read\n} # End body\nreport # Invoke",
      segments: ["function report", "{", "cat data.json", "}", "report"],
      names: ["cat", "report"],
    },
    {
      name: "attribution summary script",
      script:
        '#!/usr/bin/env bash\n# Use precomputed data\nDATA_DIR=/tmp/gh-aw/agent/community-data\n# Count authors\njq length "$DATA_DIR/attribution_by_author.json" # Count\n# Summarize issues\njq \'[.[].count] | add\' "$DATA_DIR/attribution_by_author.json" # Sum\necho done # Finished',
      segments: ["DATA_DIR=/tmp/gh-aw/agent/community-data", 'jq length "$DATA_DIR/attribution_by_author.json"', "jq '[.[].count] | add' \"$DATA_DIR/attribution_by_author.json\"", "echo done"],
      names: ["jq", "echo"],
    },
  ])("parses $name", ({ script, segments, names }) => {
    expect(splitOnPipelineOperators(script)).toEqual(segments);
    expect(extractCommandNamesFromPipeline(script)).toEqual(names);
  });
});

describe.each(lineEndings)("comment placement with $name line endings", ({ value: newline }) => {
  for (const separator of ["\n", ";", "&&", "||", "|"]) {
    it.each(commentBodies)(`ignores $name around ${JSON.stringify(separator)} separators`, ({ text }) => {
      const operator = separator === "\n" ? newline : separator;
      const script = [`# ${text}`, `cat data.json ${operator} # ${text}`, `jq length ${operator} # ${text}`, `wc -l # ${text}`, `# ${text}`].join(newline);
      expect(splitOnPipelineOperators(script)).toEqual(["cat data.json", "jq length", "wc -l"]);
      expect(extractCommandNamesFromPipeline(script)).toEqual(["cat", "jq", "wc"]);
      expect(checkPermission({ kind: "shell", fullCommandText: script })).toEqual({ kind: "approve-once" });
    });
  }

  it.each(["#", "# Header", "  # Indented", "\t# Tabbed", "#!/usr/bin/env bash", '# Unclosed "quotes && $(subshells'])("keeps comment-only requests fail-closed: %s", comment => {
    const script = `${comment}${newline}${comment}`;
    expect(splitOnPipelineOperators(script)).toEqual([]);
    expect(extractCommandNamesFromPipeline(script)).toEqual([]);
    expect(checkPermission({ kind: "shell", fullCommandText: script }).kind).toBe("reject");
  });

  it.each([
    { name: "standalone comments", prefix: "# Allowed read\ncat data.json\n# Next command is not allowed\n" },
    { name: "inline comments with unmatched quotes", prefix: 'cat data.json # Ignore this "quote and && operator\n' },
    { name: "comment ending in backslash", prefix: "cat data.json # This comment does not continue \\\n" },
    { name: "quoted hash", prefix: "echo '# literal' && " },
    { name: "multiline quoted hash", prefix: 'echo "first line\n# literal" && ' },
    { name: "escaped hash", prefix: String.raw`echo \#literal && ` },
    { name: "escaped space", prefix: String.raw`echo value\ #literal && ` },
    { name: "escaped quote", prefix: String.raw`echo \"#literal && ` },
    { name: "parameter default", prefix: "echo ${value:- # literal} && " },
    { name: "nested parameter defaults", prefix: "echo ${outer:-${inner:- # literal}} && " },
  ])("cannot conceal a disallowed command with $name", ({ prefix }) => {
    const script = `${prefix}python3 -c 'print(1)' # Final comment`.replace(/\n/g, newline);
    expect(extractCommandNamesFromPipeline(script)).toContain("python3");
    expect(checkPermission({ kind: "shell", fullCommandText: script }).kind).toBe("reject");
  });
});
