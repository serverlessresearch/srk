package integrationtest

import (
	"bytes"
	"encoding/json"
	"log"
	"path"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/lambda"

	"github.com/serverlessresearch/srk/pkg/lambdalike"
	"github.com/serverlessresearch/srk/pkg/srk"
)

func TestLocalDryRun(t *testing.T) {
	var err error
	s := lambdalike.NewApiService([]string{}, 0)
	err = s.Start()
	if err != nil {
		t.Fatal(err)
	}
	sess := session.Must(session.NewSession())
	client := lambda.New(sess, &aws.Config{
		Endpoint: aws.String("http://" + s.Addr.String()),
		Region:   aws.String("us-west-2"),
	})

	log.Printf("starting function invocation")
	resp, err := client.Invoke(&lambda.InvokeInput{
		FunctionName:   aws.String("echo"),
		Payload:        []byte{},
		InvocationType: aws.String("DryRun"),
	})
	log.Printf("finished function invocation")

	if err != nil {
		t.Fatal("Error invoking function", err)
	}
	if *resp.StatusCode != 204 {
		t.Fatalf("expected response code 204 but received %d", *resp.StatusCode)
	}
}

func TestLocalInvocation(t *testing.T) {
	s, client, err := setup()
	if err != nil {
		t.Fatal(err)
	}
	createResp, err := installObject(client, "echo")
	if err != nil {
		t.Fatal(err)
	}
	log.Print(createResp)

	type helloMessage struct {
		Message string `json:"message"`
	}
	message := "hello lambda!"

	var send = helloMessage{message}
	inputStr, err := json.Marshal(send)
	if err != nil {
		t.Fatal(err)
	}
	invokeResp, err := client.Invoke(&lambda.InvokeInput{
		FunctionName:   aws.String("echo"),
		Payload:        inputStr,
		InvocationType: aws.String("RequestResponse"),
	})

	if err != nil {
		t.Fatal("Error invoking function", err)
	}
	if *invokeResp.StatusCode != 200 {
		t.Fatalf("expected response code 200 but received %d", *invokeResp.StatusCode)
	}
	responseObj := helloMessage{}
	err = json.Unmarshal(invokeResp.Payload, &responseObj)
	if err != nil {
		t.Fatal(err)
	}
	log.Printf("response message is %q", responseObj.Message)
	if responseObj.Message != message {
		t.Fatalf("received %q but expected %q", responseObj.Message, message)
	}
	s.Shutdown()
}

func installObject(client *lambda.Lambda, name string) (*lambda.FunctionConfiguration, error) {
	var zipBytes bytes.Buffer
	codePath := path.Join("input", name)
	err := srk.ZipDirToWriter(&zipBytes, codePath, codePath)
	if err != nil {
		return nil, err
	}

	return client.CreateFunction(&lambda.CreateFunctionInput{
		FunctionName: aws.String(name),
		Runtime:      aws.String("python3.8"),
		Role:         aws.String(""),
		Handler:      aws.String("lambda_function.lambda_handler"),
		MemorySize:   aws.Int64(128),
		Timeout:      aws.Int64(3),
		Code: &lambda.FunctionCode{
			ZipFile: zipBytes.Bytes(),
		},
	})

}

func setup() (*lambdalike.ApiService, *lambda.Lambda, error) {
	s := lambdalike.NewApiService([]string{}, 0)
	err := s.Start()
	if err != nil {
		return nil, nil, err
	}
	sess := session.Must(session.NewSession())
	client := lambda.New(sess, &aws.Config{
		Endpoint: aws.String("http://" + s.Addr.String()),
		Region:   aws.String("us-west-2"),
	})
	return s, client, nil
}

func invoke(client *lambda.Lambda, functionName string, payload interface{}) (*lambda.InvokeOutput, error) {
	inputStr, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	return client.Invoke(&lambda.InvokeInput{
		FunctionName:   aws.String(functionName),
		Payload:        inputStr,
		InvocationType: aws.String("RequestResponse"),
	})
}

func TestTwoFunctions(t *testing.T) {
	s, client, err := setup()
	if err != nil {
		t.Fatal(err)
	}
	_, err = installObject(client, "echo")
	if err != nil {
		t.Fatal(err)
	}
	_, err = installObject(client, "hello")
	if err != nil {
		t.Fatal(err)
	}

	type helloMessage struct {
		Message string `json:"message"`
	}
	message := "hello lambda!"
	echoResp, err := invoke(client, "echo", &helloMessage{message})
	if err != nil {
		t.Fatal(err)
	}
	var responseObj helloMessage
	err = json.Unmarshal(echoResp.Payload, &responseObj)
	if err != nil {
		t.Fatal(err)
	}
	log.Printf("response message is %q", responseObj.Message)
	if responseObj.Message != message {
		t.Fatalf("received %q but expected %q", responseObj.Message, message)
	}

	type empty struct{}
	expectedHello := "hello world!!!"
	helloResp, err := invoke(client, "hello", &empty{})
	if err != nil {
		t.Fatal(err)
	}
	err = json.Unmarshal(helloResp.Payload, &responseObj)
	if err != nil {
		t.Fatal(err)
	}
	log.Printf("response message is %q", responseObj.Message)
	if responseObj.Message != expectedHello {
		t.Fatalf("received %q but expected %q", responseObj.Message, expectedHello)
	}
	s.Shutdown()
}
