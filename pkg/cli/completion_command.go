package cli

import (
	"bytes"
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// NewCompletionCommand creates a custom completion command that generates scripts for "gh aw" instead of "gh"
func NewCompletionCommand(rootCmd *cobra.Command) *cobra.Command {
	completionCmd := &cobra.Command{
		Use:   "completion [bash|zsh|fish|powershell]",
		Short: "Generate the autocompletion script for the specified shell",
		Long: `Generate the autocompletion script for gh aw for the specified shell.
See each sub-command's help for details on how to use the generated script.`,
		DisableFlagsInUseLine: true,
		ValidArgs:             []string{"bash", "zsh", "fish", "powershell"},
		Args:                  cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),
		Hidden:                true, // Keep hidden like the default completion command
	}

	// Add bash subcommand
	bashCmd := &cobra.Command{
		Use:   "bash",
		Short: "Generate the autocompletion script for bash",
		Long: `Generate the autocompletion script for the bash shell.

This script depends on the 'bash-completion' package.
If it is not installed already, you can install it via your OS's package manager.

To load completions in your current shell session:

	source <(gh aw completion bash)

To load completions for every new session, execute once:

#### Linux:

	gh aw completion bash > /etc/bash_completion.d/gh-aw

#### macOS:

	gh aw completion bash > $(brew --prefix)/etc/bash_completion.d/gh-aw

You will need to start a new shell for this setup to take effect.`,
		DisableFlagsInUseLine: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			var buf bytes.Buffer
			if err := rootCmd.GenBashCompletion(&buf); err != nil {
				return err
			}
			fixed := FixCompletionScript(buf.String(), "bash")
			fmt.Fprint(os.Stdout, fixed)
			return nil
		},
	}

	// Add zsh subcommand
	zshCmd := &cobra.Command{
		Use:   "zsh",
		Short: "Generate the autocompletion script for zsh",
		Long: `Generate the autocompletion script for the zsh shell.

If shell completion is not already enabled in your environment you will need
to enable it.  You can execute the following once:

	echo "autoload -U compinit; compinit" >> ~/.zshrc

To load completions in your current shell session:

	source <(gh aw completion zsh)

To load completions for every new session, execute once:

#### Linux:

	gh aw completion zsh > "${fpath[1]}/_gh-aw"

#### macOS:

	gh aw completion zsh > $(brew --prefix)/share/zsh/site-functions/_gh-aw

You will need to start a new shell for this setup to take effect.`,
		DisableFlagsInUseLine: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			var buf bytes.Buffer
			if err := rootCmd.GenZshCompletion(&buf); err != nil {
				return err
			}
			fixed := FixCompletionScript(buf.String(), "zsh")
			fmt.Fprint(os.Stdout, fixed)
			return nil
		},
	}

	// Add fish subcommand
	fishCmd := &cobra.Command{
		Use:   "fish",
		Short: "Generate the autocompletion script for fish",
		Long: `Generate the autocompletion script for the fish shell.

To load completions in your current shell session:

	gh aw completion fish | source

To load completions for every new session, execute once:

	gh aw completion fish > ~/.config/fish/completions/gh-aw.fish

You will need to start a new shell for this setup to take effect.`,
		DisableFlagsInUseLine: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			var buf bytes.Buffer
			if err := rootCmd.GenFishCompletion(&buf, true); err != nil {
				return err
			}
			fixed := FixCompletionScript(buf.String(), "fish")
			fmt.Fprint(os.Stdout, fixed)
			return nil
		},
	}

	// Add powershell subcommand
	powershellCmd := &cobra.Command{
		Use:   "powershell",
		Short: "Generate the autocompletion script for powershell",
		Long: `Generate the autocompletion script for powershell.

To load completions in your current shell session:

	gh aw completion powershell | Out-String | Invoke-Expression

To load completions for every new session, add the output of the above command
to your powershell profile.`,
		DisableFlagsInUseLine: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			var buf bytes.Buffer
			if err := rootCmd.GenPowerShellCompletionWithDesc(&buf); err != nil {
				return err
			}
			// PowerShell doesn't need the same fixes as bash/zsh
			fmt.Fprint(os.Stdout, buf.String())
			return nil
		},
	}

	completionCmd.AddCommand(bashCmd)
	completionCmd.AddCommand(zshCmd)
	completionCmd.AddCommand(fishCmd)
	completionCmd.AddCommand(powershellCmd)

	return completionCmd
}
