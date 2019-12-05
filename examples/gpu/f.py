# This is a bit of a hack for openlambda
import subprocess as sp
import sys
import ctypes

def f(event):
    N = ctypes.c_int(event['test-size'])

    cudaLib = ctypes.cdll.LoadLibrary("./handler/vadd.so")

    # C++ appeared to mangle the name of the function. I found it by running
    # "nm -D vadd.so" which lists the contents of a shared library.
    res = cudaLib._Z8cudaTesti(N)

    if res:
        return "Success\n"
    else:
        return "Failure\n"

if __name__ == "__main__":
    print(f(None))
