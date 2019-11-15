// The Berkeley serverless research toolkit

package main

import "github.com/serverlessresearch/srk/cmd/srk"

func main() {
	// This project uses cobra to handle command line arguments. Each
	// subcommand is handled by a file in cmd/.
	// https://github.com/spf13/cobra
	srk.Execute()
}
