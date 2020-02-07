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

	"github.com/pkg/errors"
	"github.com/serverlessresearch/srk/pkg/srk"
	"github.com/spf13/viper"
)

type olConfig struct {
	// Command to run base openlambda manager ('ol')
	cmd string
	// Working directory for openlambda
	dir string
	// Keeps track of if we've started an ol worker or not
	sessionStarted bool
	log            srk.Logger
}

func NewConfig(logger srk.Logger, config *viper.Viper) (srk.FunctionService, error) {
	olCfg := &olConfig{
		config.GetString("olcmd"),
		config.GetString("oldir"),
		false,
		logger,
	}
	return olCfg, nil
}

func (self *olConfig) Package(rawDir string) (string, error) {
	tarPath := filepath.Clean(rawDir) + ".tar.gz"
	rerr := tarRaw(rawDir, tarPath)
	if rerr != nil {
		return "", rerr
	}
	return tarPath, nil
}

func (self *olConfig) Install(rawDir string) error {
	tarPath := filepath.Clean(rawDir) + ".tar.gz"

	installPath := filepath.Join(self.dir, "registry", filepath.Base(tarPath))
	if err := srk.CopyFile(tarPath, installPath); err != nil {
		return err
	}
	self.log.Info("Open Lambda function installed to: " + installPath)
	return nil
}

func (self *olConfig) Remove(fName string) error {
	tarPath := filepath.Clean(fName) + ".tar.gz"

	installPath := filepath.Join(self.dir, "registry", filepath.Base(tarPath))
	if err := os.Remove(installPath); err != nil {
		return err
	}
	self.log.Info("Open Lambda function removed")
	return nil
}

func (self *olConfig) Destroy() {
	if self.sessionStarted {
		self.terminateOlWorker()
	}
}

func (self *olConfig) Invoke(fName string, args string) (resp *bytes.Buffer, rerr error) {
	if !self.sessionStarted {
		if err := self.launchOlWorker(); err != nil {
			return nil, errors.Wrap(err, "Failed to start openlambda session")
		}
	}

	olResp, err := http.Post("http://localhost:5000/run/"+fName, "application/json", strings.NewReader(args))
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
	cmd := exec.Command(self.cmd, "worker", "-d", "--path="+self.dir)
	if out, err := cmd.CombinedOutput(); err != nil {
		return errors.Wrap(err, string(out))
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
