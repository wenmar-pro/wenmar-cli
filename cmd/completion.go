package cmd

import (
	"github.com/spf13/cobra"
)

var completionCmd = &cobra.Command{
	Use:   "completion [shell]",
	Short: "Generate shell completion script",
	Long: `Generate shell autocompletion script for wenmar.

Supported shells: bash, zsh, fish, powershell.

Install:
  bash:  wenmar completion bash > /etc/bash_completion.d/wenmar
         (or ~/.local/share/bash-completion/completions/wenmar)
  zsh:   wenmar completion zsh > "${fpath[1]}/_wenmar"
  fish:  wenmar completion fish > ~/.config/fish/completions/wenmar.fish
`,
	Args:      cobra.ExactValidArgs(1),
	ValidArgs: []string{"bash", "zsh", "fish", "powershell"},
	RunE: func(cmd *cobra.Command, args []string) error {
		switch args[0] {
		case "bash":
			return cmd.Root().GenBashCompletion(cmd.OutOrStdout())
		case "zsh":
			return cmd.Root().GenZshCompletion(cmd.OutOrStdout())
		case "fish":
			return cmd.Root().GenFishCompletion(cmd.OutOrStdout(), true)
		case "powershell":
			return cmd.Root().GenPowerShellCompletionWithDesc(cmd.OutOrStdout())
		default:
			return nil
		}
	},
}

func init() {
	rootCmd.AddCommand(completionCmd)
}
