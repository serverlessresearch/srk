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

type awsLambdaConfig struct {
	// AWS arn role
	role string
	// see configs/srk.yaml for an example
	vpcConfig string
	region    string
	session   *lambda.Lambda
	log       srk.Logger
}

func NewConfig(logger srk.Logger, config *viper.Viper) (srk.FunctionService, error) {
	awsCfg := &awsLambdaConfig{
		role:      config.GetString("role"),
		vpcConfig: config.GetString("vpc-config"),
		region:    config.GetString("region"),
		session:   nil,
		log:       logger,
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

func (self *awsLambdaConfig) Session() *lambda.Lambda {

	if self.session == nil {
		sess := session.Must(session.NewSession())
		self.session = lambda.New(sess, &aws.Config{Region: aws.String(self.region)})
	}

	return self.session
}

func (self *awsLambdaConfig) Package(rawDir string) (zipDir string, rerr error) {
	zipPath := filepath.Clean(rawDir) + ".zip"
	rerr = zipRaw(rawDir, zipPath)
	if rerr != nil {
		return "", rerr
	}
	return zipPath, nil
}

func (self *awsLambdaConfig) Install(rawDir string, env map[string]string, layers []string) (rerr error) {
	zipPath := filepath.Clean(rawDir) + ".zip"
	return self.awsInstall(zipPath, env, layers)
}

func (self *awsLambdaConfig) Remove(fName string) error {

	_, err := self.Session().DeleteFunction(&lambda.DeleteFunctionInput{FunctionName: aws.String(fName)})
	if err != nil {
		return decodeAwsError(err)
	}
	return nil
}

// Install a layer to the desired FaaS service. It is assumed that
// Package() has already been called on this rawDir. The name of rawDir is
// also the name of the layer.
// Returns: ARN of the installed layer
func (self *awsLambdaConfig) InstallLayer(rawDir string, compatibleRuntimes []string) (layerId string, rerr error) {

	zipPath := filepath.Clean(rawDir) + ".zip"
	zipData, err := ioutil.ReadFile(zipPath)
	if err != nil {
		return "", errors.Wrap(err, "Failed to read the zip file we just created")
	}

	layerName := strings.TrimSuffix(filepath.Base(zipPath), ".zip")
	input := &lambda.PublishLayerVersionInput{
		LayerName:          aws.String(layerName),
		Content:            &lambda.LayerVersionContentInput{ZipFile: zipData},
		CompatibleRuntimes: aws.StringSlice(compatibleRuntimes),
	}

	self.log.Info("Uploading layer: " + layerName)
	output, err := self.Session().PublishLayerVersion(input)
	if err != nil {
		return "", decodeAwsError(err)
	}

	return *output.LayerVersionArn, nil
}

// Removes a layer from the service. Does not affect packages.
func (self *awsLambdaConfig) RemoveLayer(name string) (rerr error) {

	layerVersions, err := self.layerVersionsByName(name)
	if err != nil {
		return err
	}

	for _, layerVersion := range layerVersions {

		layerVersionARN := strings.Split(*layerVersion.LayerVersionArn, ":")
		layerARN := strings.Join(layerVersionARN[:len(layerVersionARN)-1], ":")

		input := &lambda.DeleteLayerVersionInput{
			LayerName:     aws.String(layerARN),
			VersionNumber: layerVersion.Version,
		}

		_, err := self.Session().DeleteLayerVersion(input)
		if err != nil {
			return decodeAwsError(err)
		}
	}

	return nil
}

func (self *awsLambdaConfig) layerVersionsByName(name string) ([]*lambda.LayerVersionsListItem, error) {

	var marker, layerARN *string

	for {
		output, err := self.Session().ListLayers(&lambda.ListLayersInput{Marker: marker})
		if err != nil {
			return nil, decodeAwsError(err)
		}
		for _, layer := range output.Layers {
			if *layer.LayerName == name {
				layerARN = layer.LayerArn
				break
			}
		}
		if output.NextMarker == nil {
			break
		}
		marker = output.NextMarker
	}

	if layerARN == nil {
		return nil, nil
	}

	versions := make([]*lambda.LayerVersionsListItem, 0)

	marker = nil
	for {
		input := &lambda.ListLayerVersionsInput{
			LayerName: layerARN,
			Marker:    marker,
		}

		output, err := self.Session().ListLayerVersions(input)
		if err != nil {
			return nil, decodeAwsError(err)
		}

		versions = append(versions, output.LayerVersions...)

		if output.NextMarker == nil {
			break
		}
		marker = output.NextMarker
	}

	return versions, nil
}

func (self *awsLambdaConfig) Destroy() {
	//Currently no state cleanup needed for Aws
}

func (self *awsLambdaConfig) Invoke(fName string, args string) (resp *bytes.Buffer, rerr error) {

	awsResp, err := self.Session().Invoke(&lambda.InvokeInput{
		FunctionName: aws.String(fName),
		Payload:      []byte(args),
		// This is a synchronous invocation, our API might need to change for async
		InvocationType: aws.String("RequestResponse")})
	if err != nil {
		return nil, errors.Wrap(decodeAwsError(err), "failed to invoke function")
	}
	resp = bytes.NewBuffer(awsResp.Payload)

	if awsResp.FunctionError != nil {
		return resp, errors.Wrap(errors.New(awsResp.String()), "function returned error")
	}
	self.log.Info("Function invocation success:\n")
	self.log.Infof("Executed Version: %v\n", awsResp.ExecutedVersion)
	self.log.Infof("Function Error: %v\n", awsResp.FunctionError)
	self.log.Infof("Log Result: %v\n", awsResp.LogResult)
	self.log.Infof("Payload: %s\n", string(awsResp.Payload))
	self.log.Infof("Status Code: %v\n", awsResp.StatusCode)
	return resp, nil
}

func (self *awsLambdaConfig) awsInstall(zipPath string, env map[string]string, layers []string) (rerr error) {

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

	var awsLayers []*string
	if layers != nil {
		awsLayers = aws.StringSlice(layers)
	}

	var result *lambda.FunctionConfiguration
	exists, err := lambdaExists(self.Session(), funcName)
	if err != nil {
		return errors.Wrap(err, "Failure checking function status:")
	}

	if exists {
		if awsEnv != nil {
			request := &lambda.UpdateFunctionConfigurationInput{
				FunctionName: aws.String(funcName),
				Runtime:      aws.String("provided"),
				Environment:  awsEnv,
				Layers:       awsLayers,
			}

			_, err := self.Session().UpdateFunctionConfiguration(request)
			if err != nil {
				return errors.Wrap(err, "Failure updating function configuration:")
			}
		}

		req := &lambda.UpdateFunctionCodeInput{
			FunctionName: aws.String(funcName),
			ZipFile:      zipDat,
		}

		self.log.Info("Updating Function: " + funcName)
		result, err = self.Session().UpdateFunctionCode(req)
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
			Runtime:     aws.String("provided"),
			Timeout:     aws.Int64(15),
			Environment: awsEnv,
			Layers:      awsLayers,
			VpcConfig:   &awsVpcConfig,
		}

		self.log.Info("Creating Function: " + funcName)
		result, err = self.Session().CreateFunction(req)
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
