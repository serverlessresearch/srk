// AWS Lambda specific functions. Implements the FunctionService interface.

package awslambda

import (
	"archive/zip"
	"bytes"
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
	"github.com/pkg/errors"
	"github.com/serverlessresearch/srk/pkg/srk"
	"github.com/spf13/viper"
)

type awsLambdaRuntime struct {
	base   string
	layers []string
}

type awsLambdaConfig struct {
	// AWS arn role
	role string
	// see configs/srk.yaml for an example
	vpcConfig      string
	region         string
	runtimes       map[string]awsLambdaRuntime
	defaultRuntime string
	session        *lambda.Lambda
	log            srk.Logger
}

func NewConfig(logger srk.Logger, config *viper.Viper) (srk.FunctionService, error) {

	awsCfg := &awsLambdaConfig{
		role:           config.GetString("role"),
		vpcConfig:      config.GetString("vpc-config"),
		region:         config.GetString("region"),
		runtimes:       make(map[string]awsLambdaRuntime),
		defaultRuntime: config.GetString("default-runtime"),
		session:        nil,
		log:            logger,
	}

	for name, config := range config.GetStringMap("runtimes") {

		runtimeConfig := config.(map[string]interface{})
		baseConfig := runtimeConfig["base"].(string)
		if baseConfig == "" {
			baseConfig = "provided"
		}
		layerConfig := runtimeConfig["layers"].([]interface{})

		awsCfg.runtimes[name] = awsLambdaRuntime{
			base:   baseConfig,
			layers: make([]string, len(layerConfig)),
		}

		for i := 0; i < len(layerConfig); i++ {
			awsCfg.runtimes[name].layers[i] = layerConfig[i].(string)
		}
	}

	return awsCfg, nil
}

func (self *awsLambdaConfig) ReportStats() (map[string]float64, error) {

	stats := make(map[string]float64)
	return stats, nil
}

func (self *awsLambdaConfig) ResetStats() error {
	// Nothing to reset yet
	return nil
}

func (self *awsLambdaConfig) awsSession() *lambda.Lambda {

	if self.session == nil {
		sess := session.Must(session.NewSession())
		self.session = lambda.New(sess, &aws.Config{Region: aws.String(self.region)})
	}

	return self.session
}

func (self *awsLambdaConfig) Package(rawDir string) (zipDir string, rerr error) {

	zipPath := filepath.Clean(rawDir) + ".zip"
	rerr = srk.ZipDir(rawDir, rawDir, zipPath)
	if rerr != nil {
		return "", rerr
	}
	return zipPath, nil
}

func (self *awsLambdaConfig) Install(rawDir string, env map[string]string, runtime string) (rerr error) {

	zipPath := filepath.Clean(rawDir) + ".zip"
	return self.awsInstall(zipPath, env, runtime)
}

func (self *awsLambdaConfig) Remove(fName string) error {

	_, err := self.awsSession().DeleteFunction(&lambda.DeleteFunctionInput{FunctionName: aws.String(fName)})
	if err != nil {
		return decodeAwsError(err)
	}
	return nil
}

func (self *awsLambdaConfig) Destroy() {
	//Currently no state cleanup needed for Aws
}

func (self *awsLambdaConfig) Invoke(fName string, args string) (resp *bytes.Buffer, rerr error) {

	awsResp, err := self.awsSession().Invoke(&lambda.InvokeInput{
		FunctionName: aws.String(fName),
		Payload:      []byte(args),
		// This is a synchronous invocation, our API might need to change for async
		InvocationType: aws.String("RequestResponse")})
	if err != nil {
		return nil, errors.Wrap(decodeAwsError(err), "failed to invoke function")
	}
	resp = bytes.NewBuffer(awsResp.Payload)

	if awsResp.FunctionError != nil {
		return resp, errors.Wrap(errors.New(string(awsResp.Payload)), "function returned error")
	}
	self.log.Info("Function invocation success:\n")
	self.log.Infof("Executed Version: %v\n", awsResp.ExecutedVersion)
	self.log.Infof("Function Error: %v\n", awsResp.FunctionError)
	self.log.Infof("Log Result: %v\n", awsResp.LogResult)
	self.log.Infof("Payload: %s\n", string(awsResp.Payload))
	self.log.Infof("Status Code: %v\n", awsResp.StatusCode)
	return resp, nil
}

func (self *awsLambdaConfig) awsInstall(zipPath string, env map[string]string, runtime string) (rerr error) {

	if runtime == "" {
		if self.defaultRuntime == "" {
			return errors.New("runtime needs to be specified or configured via config")
		}
		runtime = self.defaultRuntime
	}

	funcName := strings.TrimSuffix(filepath.Base(zipPath), ".zip")

	zipDat, err := ioutil.ReadFile(zipPath)
	if err != nil {
		return errors.Wrap(err, "Failed to read the zip file we just created")
	}

	var awsEnv *lambda.Environment
	if env != nil {
		vars := aws.StringMap(env)
		awsEnv = &lambda.Environment{Variables: vars}
	}

	awsLayers := []*string{}
	if runtimeConfig, exists := self.runtimes[runtime]; exists {
		if runtimeConfig.layers != nil {
			awsLayers = aws.StringSlice(runtimeConfig.layers)
		}
		if runtimeConfig.base != "" {
			runtime = runtimeConfig.base
		}
	}

	var result *lambda.FunctionConfiguration
	exists, err := lambdaExists(self.awsSession(), funcName)
	if err != nil {
		return errors.Wrap(err, "Failure checking function status:")
	}

	if exists {
		request := &lambda.UpdateFunctionConfigurationInput{
			FunctionName: aws.String(funcName),
			Runtime:      aws.String(runtime),
			Environment:  awsEnv,
			Layers:       awsLayers,
		}

		_, err := self.awsSession().UpdateFunctionConfiguration(request)
		if err != nil {
			return errors.Wrap(err, "Failure updating function configuration:")
		}

		req := &lambda.UpdateFunctionCodeInput{
			FunctionName: aws.String(funcName),
			ZipFile:      zipDat,
		}

		self.log.Info("Updating Function: " + funcName)
		result, err = self.awsSession().UpdateFunctionCode(req)
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
			Handler:      aws.String("lambda_function.lambda_handler"),
			MemorySize:   aws.Int64(3008),
			// TODO do we want to publish?
			// Publish:      aws.Bool(true),
			Role:        aws.String(self.role),
			Runtime:     aws.String(runtime),
			Timeout:     aws.Int64(15),
			Environment: awsEnv,
			Layers:      awsLayers,
			VpcConfig:   &awsVpcConfig,
		}

		self.log.Info("Creating Function: " + funcName)
		result, err = self.awsSession().CreateFunction(req)
	}
	if err != nil {
		return decodeAwsError(err)
	}

	self.log.Info("Success:", result)
	return nil
}

func lambdaExists(session *lambda.Lambda, fName string) (bool, error) {
	req := &lambda.ListFunctionsInput{}

	result, err := session.ListFunctions(req)
	if err != nil {
		return false, decodeAwsError(err)
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
			return errors.Wrap(err, "Couldn't make relative path while zipping")
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

func decodeAwsError(err error) error {

	var errStr string

	if aerr, ok := err.(awserr.Error); ok {
		errStr = fmt.Sprintln(aerr.Code(), aerr.Error())
	} else {
		errStr = fmt.Sprintln(err.Error())
	}

	return errors.New(errStr)
}
