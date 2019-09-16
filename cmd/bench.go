// Handle the "srk bench" command
package cmd

import (
	"net"

	"github.com/serverlessresearch/srk/pkg/cfbench"
	"github.com/serverlessresearch/srk/pkg/srk"
	"github.com/spf13/cobra"
)

// Filled in by cobra argument parsing in init()
var benchCmdConfig struct {
	bench        string
	functionName string
	functionArgs string
	benchParams  string
	trackingUrl  string
	logFile      string
}

// benchCmd represents the bench command
var benchCmd = &cobra.Command{
	Use:   "bench",
	Short: "Run a benchmark",
	Long: `Run the selected benchmark. You must have already created any needed
functions and configured the provider.`,
	Run: func(cmd *cobra.Command, args []string) {

		provider := getProvider()
		defer destroyProvider(provider)

		var bench srk.Benchmark
		benchArgs := srk.BenchArgs{
			benchCmdConfig.functionName,
			benchCmdConfig.functionArgs,
			benchCmdConfig.benchParams,
			benchCmdConfig.trackingUrl,
			benchCmdConfig.bench + ".out",
		}

		switch benchCmdConfig.bench {
		case "oneShot":
			var err error
			bench, err = cfbench.NewOneShot()
			if err != nil {
				panic("Failed to initialize OneShot benchmark")
			}

		case "concurrency_scan":
			panic("Concurrency scan not implemented yet")
		default:
			panic("Unrecognized benchmark: " + benchCmdConfig.bench)
		}

		if err := bench.RunBench(provider, &benchArgs); err != nil {
			panic("Benchmark Failed")
		}
		//Parse the benchmark args
		// XXX this should probably be handled by cfbench
		// var scanArgs cfbench.ConcurrencySweepArgs
		// if err := json.Unmarshal([]byte(benchCmdConfig.benchParams), &scanArgs); err != nil {
		// 	panic(err)
		// }
		//
		// var functionArgsData map[string]interface{}
		// if err := json.Unmarshal([]byte(benchCmdConfig.functionArgs), &functionArgsData); err != nil {
		// 	panic(err)
		// }
		//
		// if benchCmdConfig.trackingUrl == "" {
		// 	ip := getLocalIp()
		// 	benchCmdConfig.trackingUrl = fmt.Sprintf("http://%s:3000/", ip)
		// }
		// log.Printf("using tracking url %s", benchCmdConfig.trackingUrl)
		//
		// switch benchCmdConfig.mode {
		// case "concurrency_scan":
		// 	transitions := cfbench.GenSweepTransitions(scanArgs)
		// 	cfbench.ConcurrencySweep(benchCmdConfig.functionName,
		// 		functionArgsData,
		// 		transitions,
		// 		benchCmdConfig.trackingUrl,
		// 		benchCmdConfig.logFile)
		// default:
		// 	panic("unknown mode")
		// }

	},
}

func init() {
	rootCmd.AddCommand(benchCmd)

	benchCmd.Flags().StringVarP(&benchCmdConfig.bench, "benchmark", "b", "", "Which benchmark to run")
	benchCmd.Flags().StringVarP(&benchCmdConfig.functionName, "function-name", "n", "", "The function to run")
	benchCmd.Flags().StringVarP(&benchCmdConfig.functionArgs, "function-args", "a", "{}", "Arguments to the function")
	benchCmd.Flags().StringVarP(&benchCmdConfig.benchParams, "params", "p", "{}", "Parameters for the benchmark")
	benchCmd.Flags().StringVarP(&benchCmdConfig.trackingUrl, "trackingUrl", "u", "", "URL for posting responses")
	benchCmd.Flags().StringVarP(&benchCmdConfig.logFile, "output", "o", "log.txt", "Output File")
}

func getLocalIp() string {
	interfaces, err := net.Interfaces()
	if err != nil {
		panic(err)
	}

	var interfaceAddrs []net.IP
	for _, i := range interfaces {
		addrs, err := i.Addrs()
		if err != nil {
			panic(err)
		}
		for _, addr := range addrs {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}
			if ip != nil && !ip.IsLoopback() {
				if ip4 := ip.To4(); ip4 != nil {
					interfaceAddrs = append(interfaceAddrs, ip4)
				}
			}
		}
	}
	for _, ip := range interfaceAddrs {
		if ip[0] == 10 {
			return ip.String()
		}
	}
	for _, ip := range interfaceAddrs {
		if ip[0] == 172 && ip[1]&0xF0 == 16 {
			return ip.String()
		}
	}
	return interfaceAddrs[0].String()
}
