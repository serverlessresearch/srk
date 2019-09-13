// AWS Lambda specific functions. Implements the FaasService interface.

package awslambda

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

type AwsLambdaConfig struct {
	// AWS arn role
	role string
	// network and security settings, either '' or a valid argument to the aws
	// cli '--vpc-config' option.
	vpcConfig string
}

func NewConfig(role string, vpcConfig string) *AwsLambdaConfig {
	return &AwsLambdaConfig{
		role:      role,
		vpcConfig: vpcConfig,
	}
}

func (self *AwsLambdaConfig) Install(rawDir string) (rerr error) {
	zipPath := filepath.Clean(rawDir) + ".zip"
	rerr = zipRaw(rawDir, zipPath)
	if rerr == nil {
		fmt.Println("Created AWS Lambda zip file at: " + zipPath)
	}
	return rerr
}

func zipRaw(rawPath, dstPath string) error {
	destFile, err := os.Create(dstPath)
	if err != nil {
		return err
	}
	defer destFile.Close()

	zipWriter := zip.NewWriter(destFile)
	defer zipWriter.Close()
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

		header, err := zip.FileInfoHeader(info)
		if err != nil {
			return err
		}

		header.Name = relPath
		header.Method = zip.Deflate
		writer, err := zipWriter.CreateHeader(header)
		if err != nil {
			return err
		}

		sourceFile, err := os.Open(filePath)
		if err != nil {
			return err
		}
		_, err = io.Copy(writer, sourceFile)
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
