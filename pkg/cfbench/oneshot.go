// A simple benchmark that simply sends a single request and gets a single response
// Implements the srk.Benchmark interface
package cfbench

import (
	"io/ioutil"
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
	self.log.Infof("Invoking: %s(%s)", args.FName, args.FArgs)
	start := time.Now()
	resp, err := prov.Faas.Invoke(args.FName, args.FArgs)
	if err != nil {
		return errors.Wrapf(err, "Failed to invoke %s(%s)", args.FName, args.FArgs)
	}

	stats, err := prov.Faas.ReportStats()
	if err != nil {
		return errors.Wrapf(err, "Failed to gather statistics about %s(%s)", args.FName, args.FArgs)
	}

	if err = prov.Faas.ResetStats(); err != nil {
		return errors.Wrapf(err, "Failed to reset statistics for %s(%s)", args.FName, args.FArgs)
	}

	if len(stats) > 0 {
		self.log.Infof("Invocation statistics: \n")
		for k, v := range stats {
			self.log.Infof("%s:\t%v\n", k, v)
		}
	}

	self.log.WithFields(logrus.Fields{"time": time.Since(start)}).Infof("Function complete: %s", resp.String())

	if args.Output != "" {
		if err := ioutil.WriteFile(args.Output, resp.Bytes(), 0644); err != nil {
			return errors.Wrapf(err, "Failed to write result to %s", args.Output)
		}
		self.log.Infof("Saved result to %s", args.Output)
	}

	return nil
}
