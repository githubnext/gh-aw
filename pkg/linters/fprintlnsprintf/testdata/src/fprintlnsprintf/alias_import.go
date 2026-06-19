package fprintlnsprintf

import (
	f "fmt"
	"os"
)

// aliased import: f.Fprintln(w, f.Sprintf(...)) should still be flagged.
func flaggedAlias(name string) {
	f.Fprintln(os.Stderr, f.Sprintf("hello %s", name)) // want "use fmt.Fprintf"
}
