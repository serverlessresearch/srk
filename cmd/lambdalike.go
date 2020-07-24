// Handle the "srk lambdalike" command
package cmd

import (
	"fmt"
	"time"

	"github.com/serverlessresearch/srk/pkg/lambdalike"
	"github.com/spf13/cobra"
)

var lambdaLikeCmd = &cobra.Command{
	Use:   "lambdalike",
	Short: "Administer LambdaLike FaaS implementation",
	Long:  `Run and administer LambdaLike functionality.`,
}

var lambdaLikeAPI = &cobra.Command{
	Use:   "apiservice",
	Short: "Run the LambdaLike API Service",
	RunE: func(cmd *cobra.Command, args []string) error {

		s := lambdalike.NewApiService([]string{})
		s.Start()

		return nil
	},
}

var lambdaLikeWorker = &cobra.Command{
	Use:   "workermanager",
	Short: "Run the Worker Mananger",
	RunE: func(cmd *cobra.Command, args []string) error {

		wm := lambdalike.NewWorkerManager(nil)
		wm.Configure([]lambdalike.FunctionConfiguration{
			{
				FnName:       "echo",
				Version:      "1.0",
				Handler:      "lambda_handler",
				MemSize:      "128",
				Timeout:      "30",
				Region:       "us-west-2",
				Runtime:      "python3.8",
				ZipFileName:  "examples/echo",
				NumInstances: 2,
			},
		})
		fmt.Printf("ok now")
		time.Sleep(5 * time.Second)
		wm.Shutdown()

		return nil
	},
}

func init() {
	rootCmd.AddCommand(lambdaLikeCmd)
	lambdaLikeCmd.AddCommand(lambdaLikeAPI)
	lambdaLikeCmd.AddCommand(lambdaLikeWorker)
}
