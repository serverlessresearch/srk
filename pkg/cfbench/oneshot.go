// A simple benchmark that simply sends a single request and gets a single response
// Implements the srk.Benchmark interface
package cfbench

import (
	"github.com/pkg/errors"
	"github.com/serverlessresearch/srk/pkg/srk"
	"github.com/sirupsen/logrus"
)

type oneShotBench struct {
	log *logrus.Logger
}

// An srk.BenchFactory for the oneshot benchmark
func NewOneShot(logger *logrus.Logger) (srk.Benchmark, error) {
	return &oneShotBench{log: logger}, nil
}

func (self *oneShotBench) RunBench(prov *srk.Provider, args *srk.BenchArgs) error {
	self.log.Info("Invoking: " + args.FName + "(" + args.FArgs + ")")
	resp, err := prov.Faas.Invoke(args.FName, args.FArgs)
	if err != nil {
		return errors.Wrap(err, "Failed to invoke function "+args.FName+"("+args.FArgs+")")
	}

	self.log.Infof("Function Response:\n%v\n", resp)
	return nil
}
