package openlambda

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
)

type OlConfig struct {
	// Command to run base openlambda manager ('ol')
	cmd string
	// Working directory for openlambda
	dir string
}

func NewConfig(cmd string, dir string) *OlConfig {
	return &OlConfig{cmd, dir}
}

func (self *OlConfig) Install(rawDir string) error {
	tarPath := filepath.Clean(rawDir) + ".tar.gz"
	rerr := tarRaw(rawDir, tarPath)
	if rerr != nil {
		return rerr
	}
	fmt.Println("Created Open Lambda tar file at: " + tarPath)

	installPath := filepath.Join(self.dir, "registry", filepath.Base(tarPath))
	//OL is managed by root so we have to use sudo commands for everything
	cmd := exec.Command("sudo", "sh", "-c", "cp "+tarPath+" "+installPath)
	if out, err := cmd.CombinedOutput(); err != nil {
		fmt.Printf("Failed to install function: %v\n", err)
		fmt.Printf(string(out))
		return err
	}
	fmt.Println("Function installed to: " + installPath)
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
