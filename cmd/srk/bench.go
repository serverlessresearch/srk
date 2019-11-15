// Handle the "srk bench" command
package srk

import (
	"net"

	"github.com/pkg/errors"
	"github.com/serverlessresearch/srk/pkg/cfbench"
	"github.com/serverlessresearch/srk/pkg/srk"
	"github.com/spf13/cobra"
)

// Filled in by cobra argument parsing in init()
var benchCmdConfig struct {
	benchName    string
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
	RunE: func(cmd *cobra.Command, args []string) error {

		var bench srk.Benchmark
		benchArgs := srk.BenchArgs{
			benchCmdConfig.functionName,
			benchCmdConfig.functionArgs,
			benchCmdConfig.benchParams,
			benchCmdConfig.trackingUrl,
			benchCmdConfig.benchName + ".out",
		}

		switch benchCmdConfig.benchName {
		case "one-shot":
			var err error
			benchLogger := srkConfig.logger.WithField("module", "benchmark.one-shot")
			bench, err = cfbench.NewOneShot(benchLogger)
			if err != nil {
				return errors.Wrap(err, "Failed to initialize OneShot benchmark")
			}

		case "concurrency-scan":
			var err error
			benchLogger := srkConfig.logger.WithField("module", "benchmark.concurrency-scan")
			bench, err = cfbench.NewClient(benchLogger)
			if err != nil {
				return errors.Wrap(err, "Failed to initialize concurrency scan benchmark")
			}

		default:
			return errors.New("Unrecognized benchmark: " + benchCmdConfig.benchName)
		}

		if err := bench.RunBench(srkConfig.provider, &benchArgs); err != nil {
			return err
		}
		//Parse the benchmark args
		// XXX this is here as documentation for when we get around to implementing concurrency-sweep
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
		return nil
	},
}

func init() {
	rootCmd.AddCommand(benchCmd)

	benchCmd.Flags().StringVarP(&benchCmdConfig.benchName, "benchmark", "b", "", "Which benchmark to run")
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
