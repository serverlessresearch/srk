// AWS Lambda specific functions. Implements the FaasService interface.

package awslambda

import (
	"archive/zip"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/lambda"
)

type AwsLambdaConfig struct {
	// AWS arn role
	role string
	// see configs/srk.yaml for an example
	vpcConfig string
	region    string
}

func NewConfig(role string, vpcConfig string, region string) *AwsLambdaConfig {
	return &AwsLambdaConfig{
		role:      role,
		vpcConfig: vpcConfig,
		region:    region,
	}
}

func (self *AwsLambdaConfig) Install(rawDir string) (rerr error) {
	zipPath := filepath.Clean(rawDir) + ".zip"
	rerr = zipRaw(rawDir, zipPath)
	if rerr == nil {
		fmt.Println("Created AWS Lambda zip file at: " + zipPath)
	}

	return self.awsInstall(zipPath)
}

func (self *AwsLambdaConfig) awsInstall(zipPath string) (rerr error) {
	sess := session.Must(session.NewSession())
	client := lambda.New(sess, &aws.Config{Region: aws.String(self.region)})

	funcName := strings.TrimSuffix(filepath.Base(zipPath), ".zip")

	zipDat, err := ioutil.ReadFile(zipPath)
	if err != nil {
		panic("Failed to read the zip file we just created")
	}

	awsVpcConfig := lambda.VpcConfig{}
	if self.vpcConfig != "" {
		splitVpcConfig := strings.Split(self.vpcConfig, ",")
		awsVpcConfig.SetSecurityGroupIds([]*string{&splitVpcConfig[1]})
		awsVpcConfig.SetSubnetIds([]*string{&splitVpcConfig[0]})
	}

	cmd := &lambda.CreateFunctionInput{
		Code:         &lambda.FunctionCode{ZipFile: zipDat},
		Description:  aws.String("SRK Generated function " + funcName),
		FunctionName: aws.String(funcName),
		Handler:      aws.String("f.f"),
		MemorySize:   aws.Int64(128),
		Publish:      aws.Bool(true),
		Role:         aws.String(self.role),
		Runtime:      aws.String("python3.7"),
		Timeout:      aws.Int64(15),
		VpcConfig:    &awsVpcConfig,
	}

	result, err := client.CreateFunction(cmd)
	if err != nil {
		fmt.Println("Error while registering function with AWS:")
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case lambda.ErrCodeServiceException:
				fmt.Println(lambda.ErrCodeServiceException, aerr.Error())
			case lambda.ErrCodeInvalidParameterValueException:
				fmt.Println(lambda.ErrCodeInvalidParameterValueException, aerr.Error())
			case lambda.ErrCodeResourceNotFoundException:
				fmt.Println(lambda.ErrCodeResourceNotFoundException, aerr.Error())
			case lambda.ErrCodeResourceConflictException:
				fmt.Println(lambda.ErrCodeResourceConflictException, aerr.Error())
			case lambda.ErrCodeTooManyRequestsException:
				fmt.Println(lambda.ErrCodeTooManyRequestsException, aerr.Error())
			case lambda.ErrCodeCodeStorageExceededException:
				fmt.Println(lambda.ErrCodeCodeStorageExceededException, aerr.Error())
			default:
				fmt.Println(aerr.Error())
			}
		} else {
			// Print the error, cast err to awserr.Error to get the Code and
			// Message from an error.
			fmt.Println(err.Error())
		}
		return err
	}
	fmt.Println("Successfully registered function:")
	fmt.Println(result)

	return nil
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
