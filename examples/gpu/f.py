# This is a bit of a hack for openlambda
import subprocess as sp
import sys
import ctypes
import os

if os.path.exists('/handler'):
    vadd_obj = "/handler/vadd.so"
else:
    vadd_obj = "./vadd.so"

def f(event):
    # return "type: " + str(type(event)) + "\nValue: " + str(event['test-size'])
    N = ctypes.c_int(event['test-size'])

    # Execution in OL
    cudaLib = ctypes.cdll.LoadLibrary(vadd_obj)

    # Local execution
    # cudaLib = ctypes.cdll.LoadLibrary("./vadd.so")

    # C++ appeared to mangle the name of the function. I found it by running
    # "nm -D vadd.so" which lists the contents of a shared library.
    res = cudaLib._Z8cudaTesti(N)

    if res:
        return "Success\n"
    else:
        return "Failure\n"

if __name__ == "__main__":
    print(f({ "test-size" : 1048576 }))
