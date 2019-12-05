This example shows how to use CUDA with openlambda.

# Overview
The program does a simply vector add of two generated arrays and verifies that
the add was done correctly. The size of the vectors to add is passed in as an
argument to the cloud function.

The main included source files are:
  - f.py: the main cloud function called by openlambda
  - vadd.cu: A generic vector-addition test (basically a hello world)
  - vadd: The compiled standalone binary for vadd.cu. You can call this from the command line.
  - vadd.so: The shared-library version of vadd.cu. This can be linked into other programs and called.

f.py dynamically loads vadd.so using the ctypes library. It then converts it's
argument (a dictionary generated from the JSON arguments passed to lambda) to
native c types and invokes the cudaTest function.

# Requirements
This example requires a custom version of OpenLambda that supports GPUs:

    $ git clone -b gpu git@github.com:NathanTP/open-lambda.git

You can follow the instructions in that repo to install. You'll also need to
configure srk to use that version (see srk/README.md for more details).

# Basic usage:
First, compile the library:

    $ make vadd.so

Then, build and install the lambda to openlambda (assuming you have configured
your srk.yaml to use the GPU-enabled OpenLambda). First cd to the top of SRK and:

    $ ./srk function create -s examples/gpu

To test, you can invoke via SRK:
    $ ./srk bench -b one-shot -a '{"test-size":1048576}' -n gpu

Alternatively, you can manually run OpenLambda without SRK (see the OpenLambda documentation).

# Caveats and Gotchas
C++ apparently mangles the name of our desired function, so we have to call it as '_Z8cudaTesti'. To find out for other programs, you can use the 'nm' utility to inspect the library and look for the desired function there:

    $ nm -D LIBRARY.so


