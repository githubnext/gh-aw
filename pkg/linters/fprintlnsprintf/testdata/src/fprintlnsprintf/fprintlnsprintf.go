package fprintlnsprintf

import (
	"fmt"
	"os"
)

func flagged(name string) {
	fmt.Fprintln(os.Stderr, fmt.Sprintf("hello %s", name)) // want "use fmt.Fprintf"
}

func flaggedAlreadyNewline(name string) {
	fmt.Fprintln(os.Stderr, fmt.Sprintf("hello %s\n", name)) // want "use fmt.Fprintf"
}

func notFlagged(name string) {
	fmt.Fprintln(os.Stderr, "plain string")
	fmt.Fprintln(os.Stderr, "prefix", fmt.Sprintf("hello %s", name))
	fmt.Fprintln(os.Stderr, fmt.Sprintf("hello %s", name), "suffix")
	fmt.Fprintln(os.Stderr)
	fmt.Fprintf(os.Stderr, "hello %s\n", name)
}

func suppressed(name string) {
	//nolint:fprintlnsprintf
	fmt.Fprintln(os.Stderr, fmt.Sprintf("hello %s", name))
	fmt.Fprintln(os.Stderr, fmt.Sprintf("hello %s", name)) //nolint:fprintlnsprintf
}
