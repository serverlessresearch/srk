# This is a compatibility shim with open-lambda. OL requires that
# the file and function both be named 'f' and only includes an
# event, not a context.
import sleep_workload

def f(event):
    return lambda_handler(event, None)
