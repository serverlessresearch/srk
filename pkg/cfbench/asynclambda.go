package cfbench

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/lambda"
	"log"
)

type LambdaAsyncInvoker struct {
	session *session.Session
	client  *lambda.Lambda
}

func NewLambdaAsyncInvoker() *LambdaAsyncInvoker {
	sess := session.Must(session.NewSession())
	client := lambda.New(sess, &aws.Config{Region: aws.String("us-west-2")})
	return &LambdaAsyncInvoker{sess, client}
}

func (i *LambdaAsyncInvoker) Invoke(uuid string, experimentId string, trackingUrl string, functionName string, functionArgs map[string]interface{}) error {
	args := map[string]interface{}{
		"uuid": uuid,
		"experimentId": experimentId,
		"tracking_url": trackingUrl,
	}
	for k, v := range functionArgs {
		if _, exists := args[k]; !exists {
			args[k] = v
		} else {
			return errors.New("argument conflict")
		}
	}
	payload, err := json.Marshal(args)

	res, err := i.client.Invoke(&lambda.InvokeInput{FunctionName: aws.String(functionName), Payload: payload, InvocationType: aws.String("Event") })
	if err != nil {
		log.Fatal("Error invoking function", err)
	}
	if *res.StatusCode != 202 {
		return errors.New(fmt.Sprintf("error invoking function - status code %d", *res.StatusCode))
	}
	return nil
}