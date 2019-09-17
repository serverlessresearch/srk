// Handles the "srk function create" command

package cmd

import (
	"fmt"
	"path"
	"strings"

	"github.com/serverlessresearch/srk/pkg/srk"
	"github.com/spf13/cobra"
)

var createCmdConfig struct {
	source  string
	include string
	name    string
	service srk.FunctionService
}

var createCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new function and register it with the FaaS provider",
	Long: `This will package up your function with any needed boilerplate and
upload it to the configured FaaS provider. Create is equivalent to calling "srk
function package" and "srk function install".`,
	RunE: func(cmd *cobra.Command, args []string) error {

		createCmdConfig.service = getFunctionService()
		defer createCmdConfig.service.Destroy()

		var funcName string
		if createCmdConfig.name == "source" {
			funcName = strings.TrimSuffix(path.Base(createCmdConfig.source), path.Ext(createCmdConfig.source))
		} else {
			funcName = createCmdConfig.name
		}
		fmt.Println("Function name: " + funcName)

		includes := strings.Split(createCmdConfig.include, ",")
		rawDir := getRawPath(funcName)

		if err := createRaw(createCmdConfig.source, funcName, includes, rawDir); err != nil {
			fmt.Println("Packaging function failed: %v\n", err)
			return err
		}
		fmt.Println("Created raw function: " + rawDir)

		pkgPath, err := createCmdConfig.service.Package(rawDir)
		if err != nil {
			fmt.Printf("Packaing failed: %v\n", err)
			return err
		}
		fmt.Println("Created FaaS Package: " + pkgPath)

		if err := createCmdConfig.service.Install(rawDir); err != nil {
			fmt.Printf("Installation failed: %v\n", err)
			return err
		}
		fmt.Println("Successfully installed function")
		return nil
	},
}

func init() {
	functionCmd.AddCommand(createCmd)

	// Define the command line arguments for this subcommand
	createCmd.Flags().StringVarP(&createCmdConfig.source, "source", "s", "", "source directory or file")
	createCmd.Flags().StringVarP(&createCmdConfig.include, "include", "i", "", "what to include, e.g., bench")
	// The actual default is derived from the source option, so we set it
	// something that will be clear in the help output until we have all the
	// options parsed
	createCmd.Flags().StringVarP(&createCmdConfig.name, "function-name", "n", "source", "Optional name for this function, if different than the source name")
}
