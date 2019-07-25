# Serverless Research Kit

Serverless Research Kit (SRK) exists to accelerate research progress in serverless computing.
It aims to make it easy to innovate on services typically offered by cloud providers by including, for example,
readily hackable versions of cloud functions or cloud object storage. SRK also plans to include common benchmarks
and operational tools, so that launching and evaluating a multi-tenant serverless service is quick and easy.

## Examples

### Cloud Function Benchmark

Create a zip file containing the example workload:

```
./srk package \
  --source examples/cfbench/sleep_workload.py \
  --target sleep.zip \
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
  --zip-file fileb://$(pwd)/sleep.zip \
  --memory-size 128 \
  --vpc-config SubnetIds={{YOUR_SUBNET_IDS_HERE}},SecurityGroupIds={{YOUR_SECURITY_GROUP_IDS_HERE}}
```

In the command above you will need to make some substitutions.
`{{YOUR_ROLE_ARN_HERE}}` should look something like `arn:aws:iam::123459789012:role/service-role/my-service-role-ae04d032`,
`{{YOUR_SUBNET_IDS_HERE}}` should look something like `subnet-dd045605a058b8946,subnet-e56ceb1a1832684a4`, and
`{{YOUR_SECURITY_GROUP_IDS_HERE}}` should look something like `sg-4cf9dbb40b73ca192,sg-016fb0eb84b2f3fee`.

Now you can run a command like this to test the cloud function:

```
./srk cfbench \
  --mode concurrency_scan \
  --function-name sleepworkload \
  --function-args '{"sleep_time_ms":5000}' \
  --params '{"begin_concurrency":1,"delta_concurrency":1,"num_steps":5,"step_duration":5}'
```

You can also view the [example test function](examples/cfbench/sleep_workload.py).
