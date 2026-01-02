package cli

import "io"

// CommandProvider defines the minimal interface needed for command registration
// and shell completion setup. This allows type-safe command handling without
// importing the full cobra library in all packages.
//
// This interface is satisfied by *cobra.Command, allowing functions to accept
// command objects without a direct cobra dependency.
type CommandProvider interface {
	// GenBashCompletion generates bash completion script
	GenBashCompletion(w io.Writer) error

	// GenZshCompletion generates zsh completion script
	GenZshCompletion(w io.Writer) error

	// GenFishCompletion generates fish completion script
	// The includeDesc parameter determines whether to include command descriptions
	GenFishCompletion(w io.Writer, includeDesc bool) error
}
