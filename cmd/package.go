// Package creates a function archive that can be installed to a service.
package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"

	"github.com/serverlessresearch/srk/pkg/srk"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var packageCmdConfig struct {
	source  string
	include string
	service srk.FaasService
}

// packageCmd represents the package command
var packageCmd = &cobra.Command{
	Use:   "package",
	Short: "Package creates all the files needed to install a function, but does not actually install it.",
	Long: `Each FaaS service provider has their own format and requirements on
a code package. Typically, these take the form of an archive (e.g. .tgz or
.zip). The command will tell you where the package was saved so that you
can manually inspect or modify it.`,
	Run: func(cmd *cobra.Command, args []string) {
		packageCmdConfig.service = getFaasService()
		defer packageCmdConfig.service.Destroy()

		funcName := strings.TrimSuffix(path.Base(packageCmdConfig.source), path.Ext(packageCmdConfig.source))
		includes := strings.Split(packageCmdConfig.include, ",")
		rawDir := getRawPath(funcName)

		if err := createRaw(packageCmdConfig.source, funcName, includes, rawDir); err != nil {
			fmt.Println("Packaging function failed: %v\n", err)
			return
		}
		fmt.Println("Created raw function: " + rawDir)

		pkgPath, err := packageCmdConfig.service.Package(rawDir)
		if err != nil {
			fmt.Printf("Packaing failed: %v\n", err)
			return
		}
		fmt.Println("Package created at: " + pkgPath)
	},
}

func init() {
	functionCmd.AddCommand(packageCmd)

	// Define the command line arguments for this subcommand
	packageCmd.Flags().StringVarP(&packageCmdConfig.source, "source", "s", "", "source directory or file")
	packageCmd.Flags().StringVarP(&packageCmdConfig.include, "include", "i", "", "SRK-provided libraries to include")
}

// Returns a path to the raw directory for funcName (whether it exists or not)
func getRawPath(funcName string) string {
	return filepath.Join(
		viper.GetString("buildDir"),
		"functions",
		funcName)
}

// Place all provider-independent objects in a raw directory that will be
// packaged by the FaaS service. Will replace any existing rawDir.
// source: is the path to the user-provided source directory
// funcName: Unique name to give this function
// includes: List of standard SRK libraries to include (just the names of the packages, not paths)
// rawDir: Path where the rawDir should be made
func createRaw(source string, funcName string, includes []string, rawDir string) (err error) {
	//Shared global function build directory
	fBuildDir := filepath.Join(viper.GetString("buildDir"), "functions")
	err = os.MkdirAll(fBuildDir, os.ModeDir)
	if err != nil {
		fmt.Printf("Failed to create build directory at "+fBuildDir+": %v", err)
		return err
	}

	// Always cleanup old raw directories first
	if err := os.RemoveAll(rawDir); err != nil {
		fmt.Printf("Failed to cleanup old build directory "+rawDir+": %v", err)
		return err
	}

	cmd := exec.Command("cp", "-r", source, rawDir)
	if out, err := cmd.CombinedOutput(); err != nil {
		fmt.Printf("Adding source returned error: %v\n", err)
		fmt.Printf(string(out))
		return err
	}

	// Copy includes into the raw directory
	for _, include := range includes {
		includePath := filepath.Join(
			viper.GetString("includeDir"),
			"python",
			include)
		if _, err := os.Stat(includePath); err != nil {
			fmt.Printf("Couldn't find include: " + include)
			return err
		}
		cmd := exec.Command("cp", "-r", includePath, rawDir)
		if err := cmd.Run(); err != nil {
			fmt.Printf("Adding include returned error: %v", err)
			return err
		}
	}

	return nil
}
