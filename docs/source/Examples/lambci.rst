.. _example_lambci:

===============================================================================
How to use LambCI lambda
===============================================================================

This example shows how to setup the LambCI lambda docker container and use it
with the SRK.

*******************************************************************************
Run the the docker image
*******************************************************************************

LambCI provides docker images for various runtimes out of the box, but can also
use a custom runtime. To inject the lambda function into the container, the
``/var/task`` and ``/opt`` directories are mounted to local directories by the
`docker run` command. To be compatible with the SRK these directories need to
be inside the configured LambCI home directory and have the names ``task`` for
the lambda function and ``runtime`` for the runtime and additional layer files.

Instead of running the lambda function immediatly, SRK uses the LambCI-provided
webserver with an invocation API to execute the lambda function. Therefore the
port of the webserver has to be exposed by the ``docker run`` command.

It is also possible to inject environment variables via ``--env-file`` parameter
of ``docker run``.

Updates to the function, the runtimes files or the environment file require a
restart of the container. The helper program ``entr`` can be used to automate
this. Please see the following shell script as a loader for a LambCI lambda
container.


::

	#!/bin/sh
	  
	if [ $# -ne 3 ]; then
	        echo "Usage: ./lambci.sh <path-to-lambci-dir> <runtime-name> <function-handler>"
	        exit 1
	fi

	mkdir -p $1/task $1/runtime
	touch $1/env

	find $1/env | entr -r docker run --rm \
	  -v $1/task:/var/task:ro,delegated \
	  -v $1/runtime:/opt:ro,delegated \
	  --env-file $1/env \
	  -e DOCKER_LAMBDA_STAY_OPEN=1 \
	  -p 9001:9001 \
	  lambci/lambda:$2 \
	  $3

As an example the following command will start a Python lambda function
container with data from the ``~/lambci`` directory.

::

	$ ./lambci.sh ~/lambci python3.8 lambda_function.lambda_handler

*******************************************************************************
Execute a simple test function
*******************************************************************************

Before running a simple example the SRK configuration for LambCI lambda needs
to be set up. It is sufficient to specify the data directory and the address of
the invocation webserver as in the following example.

::

	default-provider : "lambci"

	providers :
	  lambci :
	    faas : "lambciLambda"

	service :
	 faas :
	    lambciLambda:
	      # path to the lambci directory - the following sub directories will be used:
	      # * task    directory of lambda function, /var/task in container
	      # * runtime directory of the lambda runtime, /opt in container
	      # * layers  directory of layer pool with each layer a sub directory
	      # * env     environment file for lambci docker container
	      directory : '~/lambci'
	      # address of lambci server API
	      address : 'localhost:9001'

Given that the configuration is saved to the ``configs`` directory of the SRK
project, the following commands will install the ``echo`` function to the
container and invoke it once.

::

	$ ./srk --config configs/local-srk.yaml function create -s examples/echo
	$ ./srk --config configs/local-srk.yaml bench -b one-shot -a '{"hello" : "world"}'

If everything works the function result will be ``{"hello": "world"}``.

*******************************************************************************
Adding files to the runtime
*******************************************************************************

*******************************************************************************
Using a custom runtime
*******************************************************************************


*******************************************************************************
Run the container on a remote machine
*******************************************************************************

