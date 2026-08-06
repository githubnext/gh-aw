package workflow

import "fmt"

func wrapProcessOutputForActionLog(command, logFile, title string) string {
	return fmt.Sprintf(
		"%s 2>&1 | tee -a %s | \"${GH_AW_NODE_BIN:-node}\" \"${RUNNER_TEMP}/gh-aw/actions/action_log.cjs\" %s",
		command,
		shellEscapeArg(logFile),
		shellEscapeArg(title),
	)
}
