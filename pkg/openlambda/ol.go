// Support for Open Lambda. Implements the srk.FunctionService interface.
package openlambda

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync/atomic"

	"github.com/pkg/errors"
	"github.com/serverlessresearch/srk/pkg/srk"
	"github.com/spf13/viper"
)

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
	// Keeps track of if we've started an ol worker or not
	sessionStarted bool
	log            srk.Logger
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
		cmd:            config.GetString("olcmd"),
		dir:            config.GetString("oldir"),
		urls:           urls,
		lastUrl:        0,
		isLocal:        isLocal,
		sessionStarted: false,
		log:            logger,
	}
	return olCfg, nil
}

func (self *olConfig) Package(rawDir string) (string, error) {
	if !self.isLocal {
		return "", errors.New("'Package' command is not supported in local mode")
	}
	tarPath := filepath.Clean(rawDir) + ".tar.gz"
	rerr := tarRaw(rawDir, tarPath)
	if rerr != nil {
		return "", rerr
	}
	return tarPath, nil
}

func (self *olConfig) Install(rawDir string) error {
	if !self.isLocal {
		return errors.New("'Install' command is not supported in local mode")
	}

	tarPath := filepath.Clean(rawDir) + ".tar.gz"

	installPath := filepath.Join(self.dir, "registry", filepath.Base(tarPath))
	cmd := exec.Command("/bin/cp", tarPath, installPath)
	if out, err := cmd.CombinedOutput(); err != nil {
		return errors.Wrap(err, string(out))
	}
	self.log.Info("Open Lambda function installed to: " + installPath)
	return nil
}

func (self *olConfig) Remove(fName string) error {
	if !self.isLocal {
		return errors.New("'Remove' command is not supported in local mode")
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
	if self.sessionStarted && self.isLocal {
		self.terminateOlWorker()
	}
}

func (self *olConfig) Invoke(fName string, args string) (resp *bytes.Buffer, rerr error) {
	if !self.sessionStarted {
		if err := self.launchOlWorker(); err != nil {
			return nil, errors.Wrap(err, "Failed to start openlambda session")
		}
	}

	// Round-robin between servers
	urlx := atomic.AddUint64(&self.lastUrl, 1) % uint64(len(self.urls))
	url := self.urls[urlx]
	olResp, err := http.Post(url+"/run/"+fName, "application/json", strings.NewReader(args))
	if err != nil {
		return nil, errors.Wrap(err, "Failed to POST request to ol worker")
	}
	respBuf := new(bytes.Buffer)
	respBuf.ReadFrom(olResp.Body)

	return respBuf, nil
}

// Launch the open lambda worker process in the background. Returns when the
// worker is ready to receive requests. OL does the hard work of keeping track
// of worker PIDs and stuff so we don't have to.
func (self *olConfig) launchOlWorker() error {
	self.sessionStarted = true

	//Under heavy load (e.g. many goroutines calling Invoke(), we can run out
	//of host connections. This allows us to support many more concurrent
	//connections.
	http.DefaultTransport.(*http.Transport).MaxIdleConnsPerHost = 1024

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
	self.sessionStarted = false
	return nil
}

func tarRaw(rawPath, destPath string) error {
	destFile, err := os.Create(destPath)
	if err != nil {
		return err
	}
	defer destFile.Close()

	gzw := gzip.NewWriter(destFile)
	defer gzw.Close()

	tarWriter := tar.NewWriter(gzw)
	defer tarWriter.Close()

	err = filepath.Walk(rawPath, func(filePath string, info os.FileInfo, err error) error {
		if info.IsDir() {
			return nil
		}

		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(rawPath, filePath)
		if err != nil {
			return err
		}

		header, err := tar.FileInfoHeader(info, filePath)
		if err != nil {
			return err
		}
		header.Name = relPath

		if err := tarWriter.WriteHeader(header); err != nil {
			return err
		}

		sourceFile, err := os.Open(filePath)
		if err != nil {
			return err
		}
		_, err = io.Copy(tarWriter, sourceFile)
		if err != nil {
			return err
		}

		err = sourceFile.Close()
		if err != nil {
			return err
		}
		return nil
	})

	if err != nil {
		return err
	}

	return nil
}
