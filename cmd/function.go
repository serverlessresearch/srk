// Handles the "srk function" command. This command exists solely to contain
// FaaS-specific subcommands (e.g. create, invoke, etc..)

package cmd

import (
	"github.com/spf13/cobra"
)

// functionCmd represents the function command
var functionCmd = &cobra.Command{
	Use:   "function",
	Short: "Manage FaaS function",
	Long:  `Commands for dealing with functions of your configured FaaS provider.`,
}

func init() {
	rootCmd.AddCommand(functionCmd)
}
