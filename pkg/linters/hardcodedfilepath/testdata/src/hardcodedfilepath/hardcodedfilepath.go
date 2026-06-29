package hardcodedfilepath

import (
	"fmt"
	"log"

	"constants"
)

// bad: inline path literal that matches a known constant.
func badMatchesConstant() string {
	return "/tmp/gh-aw/awf-config.json" // want `hard-coded file path.*use constant constants\.ConfigFilePath`
}

// bad: inline path that matches another known constant.
func badMatchesLogsDir() string {
	return "/tmp/gh-aw/sandbox/firewall/logs" // want `hard-coded file path.*use constant constants\.ProxyLogsDir`
}

// bad: inline path with no matching constant, should suggest extraction.
func badNoMatchingConst() string {
	return "/tmp/gh-aw/agent-stdio.log" // want `hard-coded file path.*consider extracting`
}

// bad: path in a log call (log correlation).
func badInLogCall() {
	log.Printf("reading from %s", "/tmp/gh-aw/awf-config.json") // want `hard-coded file path.*use constant constants\.ConfigFilePath`
}

// bad: path in fmt.Println (log correlation).
func badInFmtPrintln() {
	fmt.Println("/tmp/gh-aw/pre-agent-audit.txt") // want `hard-coded file path.*use constant constants\.AuditFilePath`
}

// bad: ${RUNNER_TEMP} style path with no matching constant.
func badRunnerTempPath() string {
	return "${RUNNER_TEMP}/gh-aw/mcp-servers.json" // want `hard-coded file path.*consider extracting`
}

// bad: .github/ relative path with no matching constant.
func badGitHubPath() string {
	return ".github/dependabot.yml" // want `hard-coded file path.*consider extracting`
}

// ok: using the constant directly.
func okUsingConst() string {
	return constants.ConfigFilePath
}

// ok: const declaration itself is acceptable.
const localPathConst = "/tmp/gh-aw/local-file.txt"

// ok: just a plain /tmp with no suffix (too short / generic).
func okTooShort() string {
	return "/tmp"
}

// ok: not a path at all.
func okNotAPath() string {
	return "hello world"
}

// ok: suppressed with nolint directive.
func okNolint() string {
	return "/tmp/gh-aw/suppressed.log" //nolint:hardcodedfilepath
}

// ok: path is a format string with a placeholder (not a complete path).
func okFormatVerb() string {
	return fmt.Sprintf("/tmp/gh-aw/runs/%s/output.json", "run-id")
}

// ok: path template literal with %x should be treated as a format template.
func okHexTemplateLiteral() string {
	return "/tmp/gh-aw/%x.tmp"
}

// bad: escaped %% is not a format verb and should still be reported.
func badEscapedPercentPath() string {
	return "/tmp/gh-aw/100%%-done.log" // want `hard-coded file path.*consider extracting`
}

// ok: path template with indexed width/precision/value directive (e.g. %[3]*.[2]*[1]x).
func okIndexedArgDirective() string {
	return "/tmp/gh-aw/%[3]*.[2]*[1]x"
}

// ok: very short path segment (no trailing slash after prefix).
func okShortSegment() string {
	return ".github"
}
