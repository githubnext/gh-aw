package fprintlnsprintf

import (
	"fmt"
	"os"
)

func flagged(name string) {
	fmt.Fprintln(os.Stderr, fmt.Sprintf("hello %s", name)) // want "use fmt.Fprintf"
}

func notFlagged(name string) {
	fmt.Fprintln(os.Stderr, "plain string")
	fmt.Fprintf(os.Stderr, "hello %s\n", name)
}
