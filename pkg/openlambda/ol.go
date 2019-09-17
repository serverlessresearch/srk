// Support for Open Lambda. Implements the srk.FunctionService interface.
package openlambda

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/serverlessresearch/srk/pkg/srk"
)

type olConfig struct {
	// Command to run base openlambda manager ('ol')
	cmd string
	// Working directory for openlambda
	dir string
	// Keeps track of if we've started an ol worker or not
	sessionStarted bool
}

func NewConfig(cmd string, dir string) srk.FunctionService {
	return &olConfig{cmd, dir, false}
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
	//OL is managed by root so we have to use sudo commands for everything
	cmd := exec.Command("sudo", "sh", "-c", "cp "+tarPath+" "+installPath)
	if out, err := cmd.CombinedOutput(); err != nil {
		fmt.Printf("Failed to install function: %v\n", err)
		fmt.Printf(string(out))
		return err
	}
	fmt.Println("Open Lambda function installed to: " + installPath)
	return nil
}

func (self *olConfig) Remove(fName string) error {
	tarPath := filepath.Clean(fName) + ".tar.gz"

	installPath := filepath.Join(self.dir, "registry", filepath.Base(tarPath))
	//OL is managed by root so we have to use sudo commands for everything
	cmd := exec.Command("sudo", "sh", "-c", "rm -r "+installPath)
	if out, err := cmd.CombinedOutput(); err != nil {
		fmt.Printf("Failed to remove function: %v\n", err)
		fmt.Printf(string(out))
		return err
	}
	fmt.Println("Open Lambda function removed")
	return nil

}

func (self *olConfig) Destroy() {
	if self.sessionStarted {
		self.terminateOlWorker()
	}
}

func (self *olConfig) Invoke(fName string, args string) (resp *bytes.Buffer, rerr error) {
	if !self.sessionStarted {
		self.launchOlWorker()
	}

	olResp, err := http.Post("http://localhost:5000/run/"+fName, "application/json", strings.NewReader(args))
	if err != nil {
		fmt.Printf("Failed to POST request to ol worker: %v\n", err)
		return nil, err
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
	cmd := exec.Command("sudo", "sh", "-c",
		self.cmd+" worker -d --path="+self.dir)
	if out, err := cmd.CombinedOutput(); err != nil {
		fmt.Printf("Failed to launch the open lambda worker: %v\n", err)
		fmt.Printf(string(out))
		return err
	}
	return nil
}

// Clean up the open lambda worker launched by launchOlWorker()
func (self *olConfig) terminateOlWorker() error {
	cmd := exec.Command("sudo", "sh", "-c",
		self.cmd+" kill --path="+self.dir)
	if out, err := cmd.CombinedOutput(); err != nil {
		fmt.Printf("Failed to terminate the open lambda worker: %v\n", err)
		fmt.Printf(string(out))
		return err
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
			panic(fmt.Sprintf("Couldn't make relative path while zipping %v", err))
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
