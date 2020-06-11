# Serverless Research Kit

Serverless Research Kit (SRK) exists to accelerate research progress in serverless computing.
It aims to make it easy to innovate on services typically offered by cloud providers by including, for example,
readily hackable versions of cloud functions or cloud object storage. SRK also plans to include common benchmarks
and operational tools, so that launching and evaluating a multi-tenant serverless service is quick and easy.

## Installation
SRK requires a number of runtime files in order to operate. These are all
contained in the runtime/ directory. You may optionally use the 'install.sh'
script to place these files in a global location (defaults to ~/.srk). In either case, you should
set the SRKHOME environment variable to the absolute path of whatever directory
you'd like to use for these files (you can set it to PATH/TO/SRKREPO/runtime to
always use the most up to date files when developing srk). In this case we'll
set it to the repo's version of runtime:

    $ ./install.sh
    Please specify an install location (or press enter to default to ~/.srk)
    ./runtime
    SRK installed to /PATH/TO/REPO/runtime
    Please add /PATH/TO/REPO/runtime/config.yaml and configure for your needs
    You should add "export SRKHOME=/PATH/TO/REPO/runtime" to your .bashrc or equivalent
    $ export SRKHOME=$(pwd)/runtime

You may also want to add your srk repo to PATH or copy the built binary to a
location on your path.

## Configuration
Wherever you installed SRK, you must now configure it to use different
providers. You can start by copying $SRKHOME/example-config.yaml to
$SRKHOME/config.yaml. You can then edit that file to configure your services:

* OpenLambda needs to know where you OL command and working directories are.
* AWS needs to know your role and vpc/security group configurations

You should also set the 'default-provider' to either 'aws' or 'local' (local
uses openlambda).

## Build
To build this project, just run:

  go build

## Examples

### Echo (Hello World)
The echo function is defined at examples/echo. It serves as a "Hello World" for
FaaS.

#### Source Files
* echo.py: This is our actual function logic, it can be anything you want as
  long as all of it's dependencies are in the echo/ folder that we pass to
``srk function create -s``
* f.py: This is the open-lambda glue code. OL requires that functions be named
  f() and live in f.py. The signature of f() must be preserved. The body of f()
  does any OL-specific actions and calls echo() with the correct arguments.
* aws.py: This is the aws Lambda glue code. AWS lambda is more flexible with
  function naming, but has an incompatible signature requirement (it includes a
  'context' field that OL doesn't). SRK requires that you include an aws.py
  with a function f(event, context).

#### Installation
To install to the configured service, run:

    ./srk function create --source examples/echo

#### Invocation
This example simply echos back any arguments you pass it. To do a simple
invocation, we will use the 'one-shot' benchmark that simply sends a single
request and prints a single response:

    ./srk bench \
        --bench one-shot \
        --function-args '{"hello" : "world"}' \
        --function-name echo

You should see {"hello" : "world"} printed back to you as the function response.

### Concurrency Sweep Benchmark 
______
**NOTE**
This section does not currently work (but will soon)
______

A more advanced benchmark attempts to measure the function invocation scaling
of different providers.

```
./srk function create \
  --source examples/cfbench/sleep_workload.py \
  --include cfbench
```

Install it as an AWS Lambda function:
```
aws lambda create-function \
  --function-name sleepworkload \
  --runtime python3.7 \
  --handler sleep_workload.lambda_handler \
  --timeout 10 \
  --role {{YOUR_ROLE_ARN_HERE}} \
  --zip-file fileb://$(pwd)/build/functions/sleep_workload.zip \
  --memory-size 128 \
  --vpc-config SubnetIds={{YOUR_SUBNET_IDS_HERE}},SecurityGroupIds={{YOUR_SECURITY_GROUP_IDS_HERE}}
```

In the command above you will need to make some substitutions.
`{{YOUR_ROLE_ARN_HERE}}` should look something like `arn:aws:iam::123459789012:role/service-role/my-service-role-ae04d032`,
`{{YOUR_SUBNET_IDS_HERE}}` should look something like `subnet-dd045605a058b8946,subnet-e56ceb1a1832684a4`, and
`{{YOUR_SECURITY_GROUP_IDS_HERE}}` should look something like `sg-4cf9dbb40b73ca192,sg-016fb0eb84b2f3fee`.
You may also need to set the region, e.g., with a command like `export AWS_DEFAULT_REGION=us-west-2`.

Now you can run a command like this to test the cloud function:

```
./srk bench \
  --mode concurrency-scan \
  --function-name sleepworkload \
  --function-args '{"sleep_time_ms":5000}' \
  --params '{"begin_concurrency":1,"delta_concurrency":1,"num_steps":5,"step_duration":5}'
```

You can also view the [example test function](examples/cfbench/sleep_workload.py).
