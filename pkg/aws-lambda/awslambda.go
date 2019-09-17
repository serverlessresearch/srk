// AWS Lambda specific functions. Implements the FunctionService interface.

package awslambda

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"errors"
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
	"github.com/serverlessresearch/srk/pkg/srk"
)

type awsLambdaConfig struct {
	// AWS arn role
	role string
	// see configs/srk.yaml for an example
	vpcConfig string
	region    string
	session   *lambda.Lambda
}

func NewConfig(role string, vpcConfig string, region string) srk.FunctionService {
	return &awsLambdaConfig{
		role:      role,
		vpcConfig: vpcConfig,
		region:    region,
		session:   nil,
	}
}

func (self *awsLambdaConfig) Package(rawDir string) (zipDir string, rerr error) {
	zipPath := filepath.Clean(rawDir) + ".zip"
	rerr = zipRaw(rawDir, zipPath)
	if rerr != nil {
		return "", rerr
	}
	return zipPath, nil
}

func (self *awsLambdaConfig) Install(rawDir string) (rerr error) {
	zipPath := filepath.Clean(rawDir) + ".zip"

	return self.awsInstall(zipPath)
}

func (self *awsLambdaConfig) Destroy() {
	//Currently no state cleanup needed for Aws
}

func (self *awsLambdaConfig) Invoke(fName string, args string) (resp *bytes.Buffer, rerr error) {
	if self.session == nil {
		sess := session.Must(session.NewSession())
		self.session = lambda.New(sess, &aws.Config{Region: aws.String(self.region)})
	}

	payload, err := json.Marshal(args)
	if err != nil {
		fmt.Printf("Failed to parse arguments: %v\n", err)
		return nil, err
	}

	awsResp, err := self.session.Invoke(&lambda.InvokeInput{
		FunctionName: aws.String(fName),
		Payload:      payload,
		// This is a synchronous invocation, our API might need to change for async
		InvocationType: aws.String("RequestResponse")})
	if err != nil {
		return nil, err
	}
	resp = bytes.NewBuffer(awsResp.Payload)

	if awsResp.FunctionError != nil {
		return resp, errors.New(*awsResp.FunctionError)
	}
	fmt.Println("Function invocation success:")
	fmt.Printf("Executed Version: %v\n", awsResp.ExecutedVersion)
	fmt.Printf("Function Error: %v\n", awsResp.FunctionError)
	fmt.Printf("Log Result: %v\n", awsResp.LogResult)
	fmt.Printf("Payload: %s\n", string(awsResp.Payload))
	fmt.Printf("Status Code: %v\n", awsResp.StatusCode)
	return resp, nil
}

func (self *awsLambdaConfig) awsInstall(zipPath string) (rerr error) {
	if self.session == nil {
		sess := session.Must(session.NewSession())
		self.session = lambda.New(sess, &aws.Config{Region: aws.String(self.region)})
	}

	funcName := strings.TrimSuffix(filepath.Base(zipPath), ".zip")

	zipDat, err := ioutil.ReadFile(zipPath)
	if err != nil {
		panic("Failed to read the zip file we just created")
	}

	var result *lambda.FunctionConfiguration
	exists, err := lambdaExists(self.session, funcName)
	if err != nil {
		fmt.Println("Failure checking function status:")
		return err
	}

	if exists {
		req := &lambda.UpdateFunctionCodeInput{
			FunctionName: aws.String(funcName),
			ZipFile:      zipDat}

		fmt.Println("Updating Function: " + funcName)
		result, err = self.session.UpdateFunctionCode(req)
	} else {
		awsVpcConfig := lambda.VpcConfig{}
		if self.vpcConfig != "" {
			splitVpcConfig := strings.Split(self.vpcConfig, ",")
			awsVpcConfig.SetSecurityGroupIds([]*string{&splitVpcConfig[1]})
			awsVpcConfig.SetSubnetIds([]*string{&splitVpcConfig[0]})
		}

		req := &lambda.CreateFunctionInput{
			Code:         &lambda.FunctionCode{ZipFile: zipDat},
			Description:  aws.String("SRK Generated function " + funcName),
			FunctionName: aws.String(funcName),
			Handler:      aws.String("aws.f"),
			MemorySize:   aws.Int64(128),
			Publish:      aws.Bool(true),
			Role:         aws.String(self.role),
			Runtime:      aws.String("python3.7"),
			Timeout:      aws.Int64(15),
			VpcConfig:    &awsVpcConfig,
		}

		fmt.Println("Creating Function: " + funcName)
		result, err = self.session.CreateFunction(req)
	}
	if err != nil {
		fmt.Println("Error while registering function with AWS:")
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case lambda.ErrCodeResourceConflictException:
				fmt.Println(lambda.ErrCodeResourceConflictException, aerr.Error())
			case lambda.ErrCodeServiceException:
				fmt.Println(lambda.ErrCodeServiceException, aerr.Error())
			case lambda.ErrCodeInvalidParameterValueException:
				fmt.Println(lambda.ErrCodeInvalidParameterValueException, aerr.Error())
			case lambda.ErrCodeResourceNotFoundException:
				fmt.Println(lambda.ErrCodeResourceNotFoundException, aerr.Error())
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
	fmt.Println("Success:")
	fmt.Println(result)

	return nil
}

func lambdaExists(session *lambda.Lambda, fName string) (bool, error) {
	req := &lambda.ListFunctionsInput{}

	result, err := session.ListFunctions(req)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case lambda.ErrCodeServiceException:
				fmt.Println(lambda.ErrCodeServiceException, aerr.Error())
				return false, err
			case lambda.ErrCodeTooManyRequestsException:
				fmt.Println(lambda.ErrCodeTooManyRequestsException, aerr.Error())
				return false, err
			case lambda.ErrCodeInvalidParameterValueException:
				fmt.Println(lambda.ErrCodeInvalidParameterValueException, aerr.Error())
				return false, err
			default:
				fmt.Println(aerr.Error())
				return false, err
			}
		} else {
			// Print the error, cast err to awserr.Error to get the Code and
			// Message from an error.
			fmt.Println(err.Error())
			return false, err
		}
	}

	for _, f := range result.Functions {
		if *f.FunctionName == fName {
			return true, nil
		}
	}
	return false, nil
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
