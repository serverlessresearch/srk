package main

import (
	"encoding/json"
	"flag"
	"github.com/serverlessresearch/srk/pkg/cfbench"
	"os"
)

// We structure the SRK command line tool as a single executable. We do this in hopes of bringing some cohesion to
// what could otherwise easily become a highly fragmented set of utilities. We use the subcommand pattern
// (http://blog.ralch.com/tutorial/golang-subcommands/) as is common for many cloud utilities.
func main() {
	// TODO this is a very basic implementation. Among other things improve error handling.
	if len(os.Args) < 2 {
		panic("not enough arguments")
	}

	switch os.Args[1] {
	case "bench":
		flags := flag.NewFlagSet("bench", flag.ContinueOnError)
		mode := flags.String("mode", "", "Mode of benchmark")
		functionName := flags.String("function-name", "", "Which function to run")
		functionArgs := flags.String("function-args", "{}", "Arguments to the function")
		benchParams := flags.String("params", "{}", "JSON arguments")
		logfile := flags.String("output", "log.txt", "Output file")

		if err := flags.Parse(os.Args[2:]); err != nil {
			panic(err)
		}

		var scanArgs cfbench.ConcurrencySweepArgs
		if err := json.Unmarshal([]byte(*benchParams), &scanArgs); err != nil {
			panic(err)
		}

		var functionArgsData map[string]interface{}
		if err := json.Unmarshal([]byte(*functionArgs), &functionArgsData); err != nil {
			panic(err)
		}

		switch *mode {
		case "concurrency_scan":
			transitions := cfbench.GenSweepTransitions(scanArgs)
			cfbench.ConcurrencySweep(*functionName, functionArgsData, transitions, *logfile)
		default:
			panic("unknown mode")
		}
	default:
		panic("unknown command")
	}
}
