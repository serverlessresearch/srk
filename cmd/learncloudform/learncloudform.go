package main

import (
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudformation"
	cft "github.com/awslabs/goformation/v3/cloudformation"
	"github.com/awslabs/goformation/v3/cloudformation/ec2"
	"github.com/awslabs/goformation/v3/cloudformation/tags"
	"log"
	"os"
)

const setupScript = `#!/bin/bash

yum update -y
yum install -y git
wget -O - https://dl.google.com/go/go1.13.4.linux-amd64.tar.gz | tar xz -C /usr/local
ln -s /usr/local/go/bin/go /usr/local/bin/go
go get github.com/serverlessresearch/srk/
cd 
`

func buildTemplate() ([]byte, error) {
	template := cft.NewTemplate()
	server := ec2.Instance{
		//AvailabilityZone:    "",
		//BlockDeviceMappings: nil,
		//CpuOptions: &ec2.Instance_CpuOptions{
		//	CoreCount:      0,
		//	ThreadsPerCore: 0,
		//},
		//CreditSpecification: &ec2.Instance_CreditSpecification{
		//	CPUCredits: "",
		//},
		//EbsOptimized:                      true,
		ImageId:                           "ami-0a85857bfc5345c38",
		InstanceInitiatedShutdownBehavior: "",
		InstanceType:                      "t2.micro",
		KeyName:                           "serverless",
		SecurityGroupIds:                  []string{"sg-011bc68753d133d2b"},
		Tags:                              []tags.Tag{tags.Tag{Key: "Name", Value: "controlserver"}},
		UserData:                          "",
		Volumes:                           nil,
	}
	template.Resources["myserver"] = &server
	return template.JSON()
}


// sudo sed -i 's/^#UseDNS yes$/UseDNS no/' /etc/ssh/sshd_config
// sudo /bin/systemctl restart sshd.service
// wget -O - https://dl.google.com/go/go1.13.4.linux-amd64.tar.gz | tar xz -C /usr/local
// ln -s /usr/local/go/bin/go
func main() {
	sess := session.Must(session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable,
	}))

	// Create CloudFormation client in region
	templateText, err := buildTemplate()
	if err != nil {
		log.Fatal(err)
	}

	stackName := "mystack2"
	svc := cloudformation.New(sess)

	//cloudformation.DescribeStacksInput{}
	dso, err := svc.DescribeStacks(&cloudformation.DescribeStacksInput{StackName: aws.String(stackName)})
	if err != nil {
		fmt.Print(err)
	}
	fmt.Print(dso)

	input := &cloudformation.CreateStackInput{TemplateBody: aws.String(string(templateText)), StackName: aws.String(stackName)}
	cso, err := svc.CreateStack(input)
	if err != nil {
		panic(err)
	}
	fmt.Printf("have stack with id %s\n", *cso.StackId)

	desInput := &cloudformation.DescribeStacksInput{StackName: aws.String(stackName)}
	err = svc.WaitUntilStackCreateComplete(desInput)
	if err != nil {
		fmt.Println("Got error waiting for stack to be created")
		fmt.Println(err)
		os.Exit(1)
	}

	fmt.Println("Created stack " + stackName)

}
