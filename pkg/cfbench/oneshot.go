// A simple benchmark that simply sends a single request and gets a single response
// Implements the srk.Benchmark interface
package cfbench

import (
	"time"

	"github.com/pkg/errors"
	"github.com/serverlessresearch/srk/pkg/srk"
	"github.com/sirupsen/logrus"
)

type oneShotBench struct {
	log logrus.FieldLogger
}

// An srk.BenchFactory for the oneshot benchmark
func NewOneShot(logger srk.Logger) (srk.Benchmark, error) {
	return &oneShotBench{log: logger}, nil
}

func (self *oneShotBench) RunBench(prov *srk.Provider, args *srk.BenchArgs) error {
	self.log.Info("Invoking: " + args.FName + "(" + args.FArgs + ")")
	start := time.Now()
	resp, err := prov.Faas.Invoke(args.FName, args.FArgs)
	if err != nil {
		return errors.Wrap(err, "Failed to invoke function "+args.FName+"("+args.FArgs+")")
	}

	stats, err := prov.Faas.ReportStats()
	if err != nil {
		return errors.Wrap(err, "Failed to gather statistics about function "+args.FName+"("+args.FArgs+")")
	}

	if err = prov.Faas.ResetStats(); err != nil {
		return errors.Wrap(err, "Failed to reset statistics for function "+args.FName+"("+args.FArgs+")")
	}

	self.log.Infof("Invocation statistics: \n")
	for k, v := range stats {
		self.log.Infof("%s:\t%v\n", k, v)
	}

	time := time.Since(start)
	self.log.Infof("Function Complete. Took %v\n", time)
	self.log.Infof("Function Response:\n%v\n", resp)
	return nil
}
