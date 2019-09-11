// Handles the "srk function create" command

package cmd

import (
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/serverlessresearch/srk/pkg/cfpackage"
	"github.com/spf13/cobra"
)

var createCmdConfig struct {
	source  string
	include string
}

var createCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new function and register it with the FaaS provider",
	Long: `This will package up your function with any needed boilerplate and
upload it to the configured FaaS provider.`,
	Run: func(cmd *cobra.Command, args []string) {
		includes := strings.Split(createCmdConfig.include, ",")
		target := "./build/functions/" +
			strings.TrimSuffix(path.Base(createCmdConfig.source), path.Ext(createCmdConfig.source)) +
			".zip"

		// Make the build directory if needed (users can clean the system by simply 'rm -r build')
		os.MkdirAll(filepath.Dir(target), os.ModePerm)

		if err := cfpackage.Package(source, target, includes); err != nil {
			panic(err)
		}
	},
}

func init() {
	functionCmd.AddCommand(createCmd)

	createCmd.Flags().StringVarP(&createCmdConfig.source, "source", "s", "", "source directory or file")
	createCmd.Flags().StringVarP(&createCmdConfig.include, "include", "i", "", "what to include, e.g., bench")
}
