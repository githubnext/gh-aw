package cli

import (
	"fmt"
	"os"
	"time"

	"github.com/github/gh-aw/pkg/console"
)

func reportUpdateNetworkOperation(operation string) func(error) {
	fmt.Fprintln(os.Stderr, console.FormatVerboseMessage("Network: starting "+operation))
	start := time.Now()
	return func(err error) {
		status := "succeeded"
		if err != nil {
			status = "failed"
		}
		fmt.Fprintln(os.Stderr, console.FormatVerboseMessage(fmt.Sprintf("Network: %s took %s (%s)", operation, time.Since(start).Round(time.Millisecond), status)))
	}
}
