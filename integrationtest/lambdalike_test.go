package integrationtest

import (
	"encoding/json"
	"log"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/lambda"

	"github.com/serverlessresearch/srk/pkg/lambdalike"
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
		Endpoint: aws.String("http://" + s.Addr),
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
	var err error
	s := lambdalike.NewApiService([]string{}, 0)
	err = s.Start()
	if err != nil {
		t.Fatal(err)
	}
	sess := session.Must(session.NewSession())
	client := lambda.New(sess, &aws.Config{
		Endpoint: aws.String("http://" + s.Addr),
		Region:   aws.String("us-west-2"),
	})

	s.InstallFunction(lambdalike.FunctionConfiguration{
		FnName:      "echo",
		Version:     "",
		Handler:     "lambda_function.lambda_handler",
		MemSize:     "128",
		Region:      "us-west-2",
		Runtime:     "python3.8",
		ZipFileName: "examples/echo",
	})

	type helloMessage struct {
		Message string `json:"message"`
	}
	message := "hello lambda!"

	var send = helloMessage{message}
	inputStr, err := json.Marshal(send)
	if err != nil {
		t.Fatal(err)
	}
	resp, err := client.Invoke(&lambda.InvokeInput{
		FunctionName:   aws.String("echo"),
		Payload:        inputStr,
		InvocationType: aws.String("RequestResponse"),
	})

	if err != nil {
		t.Fatal("Error invoking function", err)
	}
	if *resp.StatusCode != 200 {
		t.Fatalf("expected response code 200 but received %d", *resp.StatusCode)
	}
	responseObj := helloMessage{}
	err = json.Unmarshal(resp.Payload, &responseObj)
	if err != nil {
		t.Fatal(err)
	}
	if responseObj.Message != message {
		t.Fatalf("received %q but expected %q", responseObj.Message, message)
	}
	s.Shutdown()
}
