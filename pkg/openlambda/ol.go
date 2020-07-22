// Support for Open Lambda. Implements the srk.FunctionService interface.
package openlambda

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync/atomic"
	"time"

	"github.com/pkg/errors"
	"github.com/serverlessresearch/srk/pkg/srk"
	"github.com/spf13/viper"
)

type olStats struct {
	tInvoke int64
	nInvoke int64
}

type olConfig struct {
	// Command to run base openlambda manager ('ol')
	cmd string
	// Working directory for openlambda
	dir string
	// List of URL (including port) to send invocations to
	urls []string
	// Atomically increments for every message sent (urls[lastUrl % len(urls)] is the last used URL)
	lastUrl uint64
	// Tracks whether we are interacting with a local OL server or remote
	isLocal bool
	log     srk.Logger
	stats   olStats
}

func NewConfig(logger srk.Logger, config *viper.Viper) (srk.FunctionService, error) {
	if !config.IsSet("olservers") {
		return nil, errors.New("Option 'olservers' is required")
	}
	urls := config.GetStringSlice("olservers")
	isLocal := (len(urls) == 1 && urls[0][:16] == "http://localhost")

	if isLocal && !(config.IsSet("olcmd") && config.IsSet("oldir")) {
		return nil, errors.New("Options 'olcmd' and 'oldir' are required in local mode")
	}

	olCfg := &olConfig{
		cmd:     config.GetString("olcmd"),
		dir:     config.GetString("oldir"),
		urls:    urls,
		lastUrl: 0,
		isLocal: isLocal,
		log:     logger,
		stats:   olStats{0, 0},
	}

	if err := olCfg.launchOlWorker(); err != nil {
		return nil, errors.Wrap(err, "Failed to start openlambda session")
	}

	return olCfg, nil
}

func (self *olConfig) ReportStats() (map[string]float64, error) {
	olResp, err := http.Post(self.urls[0]+"/stats", "application/json", strings.NewReader(""))
	if err != nil {
		return nil, errors.Wrap(err, "Failed to POST stats request to ol worker")
	}

	respBuf, err := ioutil.ReadAll(olResp.Body)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to read stats response")
	}

	stats := make(map[string]float64)

	var rawDecoded interface{}
	if err = json.Unmarshal(respBuf, &rawDecoded); err != nil {
		return nil, errors.Wrap(err, "Failed to interpret OL statistics")
	}

	decoded := rawDecoded.(map[string]interface{})
	for k, v := range decoded {
		if floatv, ok := v.(float64); ok {
			stats[k] = floatv
		} else {
			self.log.Warnf("Ignoring non-numeric statistics result from openLambda: %v=%v", k, v)
		}
	}
	stats["srkInvoke"] = (float64)(self.stats.tInvoke) / (float64)(self.stats.nInvoke)

	return stats, nil
}

func (self *olConfig) ResetStats() error {
	//Reset the statistics
	_, err := http.Post(self.urls[0]+"/stats", "application/json", strings.NewReader("reset"))
	if err != nil {
		return err
	}
	self.stats.tInvoke = 0
	self.stats.nInvoke = 0
	return nil
}

func (self *olConfig) Package(rawDir string) (string, error) {
	if !self.isLocal {
		return "", errors.New("'Package' command is only supported in local mode")
	}
	tarPath := filepath.Clean(rawDir) + ".tar.gz"
	rerr := srk.TarDir(rawDir, rawDir, tarPath)
	if rerr != nil {
		return "", rerr
	}
	return tarPath, nil
}

func (self *olConfig) Install(rawDir string) error {
	if !self.isLocal {
		return errors.New("'Install' command is only supported in local mode")
	}

	tarPath := filepath.Clean(rawDir) + ".tar.gz"

	installPath := filepath.Join(self.dir, "registry", filepath.Base(tarPath))
	if err := srk.CopyFile(tarPath, installPath); err != nil {
		return err
	}
	self.log.Info("Open Lambda function installed to: " + installPath)
	return nil
}

func (self *olConfig) Remove(fName string) error {
	if !self.isLocal {
		return errors.New("'Remove' command is only supported in local mode")
	}

	tarPath := filepath.Clean(fName) + ".tar.gz"

	installPath := filepath.Join(self.dir, "registry", filepath.Base(tarPath))
	if err := os.Remove(installPath); err != nil {
		return err
	}
	self.log.Info("Open Lambda function removed")
	return nil
}

func (self *olConfig) Destroy() {
	if self.isLocal {
		self.terminateOlWorker()
	}
}

func (self *olConfig) Invoke(fName string, args string) (resp *bytes.Buffer, rerr error) {
	// Round-robin between servers
	urlx := atomic.AddUint64(&self.lastUrl, 1) % uint64(len(self.urls))
	url := self.urls[urlx]
	start := time.Now()
	olResp, err := http.Post(url+"/run/"+fName, "application/json", strings.NewReader(args))
	if err != nil {
		return nil, errors.Wrap(err, "Failed to POST request to ol worker")
	}
	respBuf := new(bytes.Buffer)
	respBuf.ReadFrom(olResp.Body)

	atomic.AddInt64(&self.stats.tInvoke, time.Since(start).Microseconds())
	atomic.AddInt64(&self.stats.nInvoke, 1)

	return respBuf, nil
}

// Launch the open lambda worker process in the background. Returns when the
// worker is ready to receive requests. OL does the hard work of keeping track
// of worker PIDs and stuff so we don't have to.
func (self *olConfig) launchOlWorker() error {
	//Under heavy load (e.g. many goroutines calling Invoke(), we can run out
	//of host connections. This allows us to support many more concurrent
	//connections.
	http.DefaultTransport.(*http.Transport).MaxIdleConnsPerHost = 0

	if self.isLocal {
		cmd := exec.Command(self.cmd, "worker", "-d", "--path="+self.dir)
		if out, err := cmd.CombinedOutput(); err != nil {
			return errors.Wrap(err, string(out))
		}
	}
	return nil
}

// Clean up the open lambda worker launched by launchOlWorker()
func (self *olConfig) terminateOlWorker() error {
	cmd := exec.Command(self.cmd, "kill", "--path="+self.dir)
	if out, err := cmd.CombinedOutput(); err != nil {
		return errors.Wrap(err, "Failed to terminate the open lambda worker:\n"+string(out))
	}
	return nil
}
