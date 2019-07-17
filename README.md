# Serverless Research Kit

Serverless Research Kit (SRK) exists to accelerate research progress in serverless computing.
It aims to make it easy to innovate on services typically offered by cloud providers by including, for example,
readily hackable versions of cloud functions or cloud object storage. SRK also plans to include common benchmarks
and operational tools, so that launching and evaluating a multi-tenant serverless service is quick and easy.

## Examples

### Cloud Function Benchmark

Here is one command that you can run to test a cloud function:

```
srk bench \
  --mode concurrency_scan \
  --function-name myfunction \
  --params='{"begin_concurrency":1,"end_concurrency":10,"num_steps":5,"step_duration":5}'
```

See an [example test function](examples/cfbench/sleep_workload.py).
