package main

import (
	"encoding/base64"
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudformation"
	cft "github.com/awslabs/goformation/v3/cloudformation"
	"github.com/awslabs/goformation/v3/cloudformation/ec2"
	"github.com/awslabs/goformation/v3/cloudformation/tags"
	"github.com/serverlessresearch/srk/pkg/srk"
	"log"
	"os"
)

const setupScript = `#!/bin/bash

sed -i 's/^#UseDNS yes$/UseDNS no/' /etc/ssh/sshd_config
/bin/systemctl restart sshd.service

yum update -y
yum install -y git

mkdir /home/ec2-user/config
echo -n '%s' > /home/ec2-user/config/server.crt
echo -n '%s' > /home/ec2-user/config/server.key

chown -R ec2-user.ec2-user /home/ec2-user

wget --quiet -O - https://dl.google.com/go/go1.13.4.linux-amd64.tar.gz | tar xz -C /usr/local
ln -s /usr/local/go/bin/go /usr/local/bin/go
#export GOPATH=/home/ec2-user/go
#mkdir $GOPATH

sudo -u ec2-user bash -c '/usr/local/bin/go get -v -d github.com/serverlessresearch/srk/;\
    cd /home/ec2-user/go/src/github.com/serverlessresearch/srk/;\
    git checkout benchmark;\
    /usr/local/bin/go install ./...'
sudo -u ec2-user bash -c 'cd /home/ec2-user; nohup /home/ec2-user/go/bin/benchcontrol 2>&1 > benchcontrol.log&'
`

func addIp(t *cft.Template) {
	t.Resources["eip"] = &ec2.EIP{}
}

func addServer(t *cft.Template) error {
	cert, key, err := srk.CreateServerKeyPair([]string{"*.us-west-2.compute.amazonaws.com"})
	if err != nil {
		return err
	}
	fmt.Printf("have certificate %s\n", string(cert))
	fmt.Printf("have key %s\n", string(key))


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
		UserData:                          base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf(setupScript, cert, key))),
		Volumes:                           nil,
	}
	t.Resources["myserver"] = &server
	return nil
}


// sudo sed -i 's/^#UseDNS yes$/UseDNS no/' /etc/ssh/sshd_config
// sudo /bin/systemctl restart sshd.service
// wget -O - https://dl.google.com/go/go1.13.4.linux-amd64.tar.gz | tar xz -C /usr/local
// ln -s /usr/local/go/bin/go
func main() {
	sess := session.Must(session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable,
	}))

	template := cft.NewTemplate()
	//addIp(template)
	err := addServer(template)
	if err != nil {
		panic(err)
	}

	stackName := "mystack13"
	svc := cloudformation.New(sess)

	//cloudformation.DescribeStacksInput{}
	dso, err := svc.DescribeStacks(&cloudformation.DescribeStacksInput{StackName: aws.String(stackName)})
	if err != nil {
		fmt.Print(err)
	}
	fmt.Print(dso)

	// Create CloudFormation client in region
	templateText, err := template.JSON()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Print(string(templateText))
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

	//// Now get the ip address
	//template.addServer()


	fmt.Println("Created stack " + stackName)

}
