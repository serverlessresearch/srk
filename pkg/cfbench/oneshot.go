// A simple benchmark that simply sends a single request and gets a single response
// Implements the srk.Benchmark interface
package cfbench

import (
	"fmt"

	"github.com/serverlessresearch/srk/pkg/srk"
)

type oneShotBench struct {
}

// An srk.BenchFactory for the oneshot benchmark
func NewOneShot() (srk.Benchmark, error) {
	return &oneShotBench{}, nil
}

func (ctx *oneShotBench) RunBench(prov *srk.Provider, args *srk.BenchArgs) error {
	fmt.Println("Invoking: " + args.FName + "(" + args.FArgs + ")")
	resp, err := prov.Faas.Invoke(args.FName, args.FArgs)
	if err != nil {
		fmt.Printf("Failed to invoke function "+args.FName+"("+args.FArgs+"): %v\n", err)
		return err
	}

	fmt.Println("Function Response:")
	fmt.Printf("%v\n", resp)

	return nil
}
