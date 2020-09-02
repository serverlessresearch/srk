# Instructions for using SRK LambdaLike

## Overview

The LambdaLike tool within SRK provides a scalable and self-hosted version of AWS
Lambda that is compatible with the AWS API, including clients such as [Boto3](https://boto3.amazonaws.com/v1/documentation/api/latest/reference/services/lambda.html) and the [AWS command line tools](https://awscli.amazonaws.com/v2/documentation/api/latest/reference/lambda/index.html).

Presently, we have a configuration.
There is one *API Service* which provides the http endpoint for clients and which maintains a work queue.
In addition, you may launch one or more *Worker Managers*, which provide function runtime environments and allow scalability.
The API Service also has the ability to run an embedded Worker Manager, allowing LambdaLike to run as a single process.

## Local example with embedded Worker Manager

Start up the API Service

```
srk lambdalike apiservice
```

Continuing in a separate window, create an echo zip file
```
(rm echo.zip && cd examples/echo; zip -r ../../echo.zip lambda_function.py echo.py)
```

Create the function in LambdaLike.
```
aws lambda create-function --endpoint http://localhost:9001 --no-sign-request \
    --function-name echo --runtime python3.8 --role myrole --handler lambda_function.lambda_handler \
    --zip-file fileb://echo.zip
```
Note that you must set the `AWS_DEFAULT_REGION` environment variable if you have not already done so.

Now you can invoke the function:
```
aws lambda invoke --endpoint http://localhost:9001 --no-sign-request \
    --cli-binary-format raw-in-base64-out \
    --function-name echo --payload '{"message": "hi there!"}' output.json
```
If you are using a version 1 AWS CLI then omit `--cli-binary-format raw-in-base64-out`.

Check the output:
```
% cat echo_output.json 
{"message": "hi there!"}
```

## Local example with separate Worker Manager

Start up two worker managers in the background

```
srk lambdalike workermanager --address localhost:8001 & \
srk lambdalike workermanager --address localhost:8002 &
```

Start up the API Service
```
srk lambdalike apiservice --workers localhost:8001,localhost:8002
```

You can now install and run the echo function using the same commands as for the embedded Worker Manager.

*This step is not fully implemented:*
Concurrency for each function is configured statically.
The API Service will distribute function runtime instances among the Worker Managers.
```
aws lambda put-function-concurrency --endpoint http://localhost:9001 --no-sign-request \
    --function-name echo --reserved-concurrent-executions 10
```

# Using with GPUs

Build the Docker LambdaLike GPU Docker image. The one provided here is for Python 3.8.

```
docker build -t lambci/lambda:python3.8-cuda gpuimg/python3.8
```

To use this image you will need a machine that has GPUs.
We have run experiments using AWS p2.xlarge instances with the 
Deep Learning AMI (Amazon Linux 2, Version 30.1).

This Docker image combines the functionality of Docker Lambda (see [GitHub](https://github.com/lambci/docker-lambda) and [Docker Hub](https://hub.docker.com/r/lambci/lambda/)) with that of the [NVIDIA Container Toolkit](https://github.com/NVIDIA/nvidia-docker).

You can invoke a function directly, e.g.,
```
cd examples/echo
docker run --rm -v "$PWD":/var/task:ro,delegated \
    --gpus all \
    lambci/lambda:python3.8-cuda \
    localtest.lambda_handler '{}'
```

You can also use the `python3.8-cuda` image with LambdaLike.
Just be sure to build the Docker image on the machine that is running the Worker Manager (possibly embedded with the API Service).

You can run the echo program with GPU support as follows:
```
aws lambda create-function --endpoint http://localhost:9001 --no-sign-request \
    --function-name echo --runtime python3.8 --role myrole --handler lambda_function.lambda_handler \
    --zip-file fileb://echo.zip
```
