# This is a bit of a hack for openlambda
import subprocess as sp
import sys
import ctypes
import os

if os.path.exists('/handler'):
    vadd_obj = "/handler/vadd.so"
elif os.path.exists('/var/task'):
    vadd_obj = "/var/task/vadd.so"
else:
    vadd_obj = "./vadd.so"

def f(event):
    try:
        N = ctypes.c_int(event['test-size'])

        # Execution in OL
        cudaLib = ctypes.cdll.LoadLibrary(vadd_obj)

        # Local execution
        # cudaLib = ctypes.cdll.LoadLibrary("./vadd.so")

        # C++ appeared to mangle the name of the function. I found it by running
        # "nm -D vadd.so" which lists the contents of a shared library.
        res = cudaLib.cudaTest(N)
    except Exception as e:
        return {
                "success" : False,
                "error" : str(e)
            }

    if res:
        return {
                "success" : True,
                "error" : ""
               }
    else:
        return {
                "success" : False,
                "error" : "cuda library failed"
               }

def aws_handler(event, ctx):
    return f(event)

if __name__ == "__main__":
    print(f({ "test-size" : 1048576 }))
