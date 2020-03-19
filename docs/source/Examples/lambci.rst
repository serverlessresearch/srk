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
``docker run`` command. To be compatible with the SRK these directories need to
be inside the configured LambCI home directory and have the names ``task`` for
the lambda function and ``runtime`` for a custom runtime or additional layer
files.

Instead of running the lambda function immediatly, SRK uses the LambCI-provided
webserver with an invocation API to execute the lambda function. Therefore the
port of the webserver has to be exposed by the ``docker run`` command.

It is also possible to inject environment variables via the ``--env-file``
parameter of ``docker run``.

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

Files in the ``runtime`` directory can be found in the ``/opt`` directory of
the docker container. This is useful to add libraries and other files to the
lambda runtime environment independent of the function code.

The ``runtime`` directory (as the ``task`` directory) is managed by the SRK,
it is not possible to directly places files here. Instead, files have to be
provided via a sub directory in the ``layers`` directory. The name of the sub
directory then is the name of the layer.

E.g. to add a python package, create a layer subdirectory in the ``layers``
directory, create a directory ``python`` in it and download the required
package. The following example shows how to provide the
`request <https://requests.readthedocs.io/en/master/>`_ package.

::

	$ cd ~/lambci/layers
	$ mkdir -p requests/python && cd requests/python
	$ curl -L https://github.com/psf/requests/tarball/master | tar xz
	$ mv psf-requests* requests

The layer then can be added to a runtime via an additional configuration, as in
the following example that configures an additional ``requests`` layer.

::

	default-provider : "lambci"

	providers :
	  lambci :
	    faas : "lambciLambda"

	service :
	 faas :
	    lambciLambda:
	      # path to the lambci directory
	      directory : '~/lambci'
	      # address of lambci server API
	      address : 'localhost:9001'
	      # runtime configuration
	      runtimes :
	        # with python requests package
	        with-requests :
	          # list of additional layers 
	          layers :
	            - 'requests'

The ``with-request`` runtime can then be specified at function creation. SRK
will configure the ``lambci/runtime`` directory to contain the ``requests``
layer.

::

	$ ./srk --config configs/local-srk.yaml function create -s examples/echo -r with-requests


*******************************************************************************
Using a custom runtime
*******************************************************************************

To use a custom runtime, specify ``provided`` as runtime name for the
``lambci.sh`` script. The lambda container then expects a complete lambda
runtime in the ``runtime`` directory. For this, create a layer that contains
the runtime code and configure it in the configuration.

::

	default-provider : "lambci"

	providers :
	  lambci :
	    faas : "lambciLambda"

	service :
	 faas :
	    lambciLambda:
	      # path to the lambci directory
	      directory : '~/lambci'
	      # address of lambci server API
	      address : 'localhost:9001'
	      # runtime configuration
	      runtimes :
	        # with python requests package
	        custom-runtime :
	          # list of additional layers 
	          layers :
	            - 'custom-python'
	            - 'requests'

The custom runtime can be specified at function creation. In the example above,
SRK will copy the contents of the ``custom-python`` directory (the custom
runtime) and the ``requests`` layer to the ``runtime`` directory so that the
LambCI ``provided`` container finds it in ``/opt``.

::

	$ ./srk --config configs/local-srk.yaml function create -s examples/echo -r custom-runtime


*******************************************************************************
Run the container on a remote machine
*******************************************************************************

For certain experiments it is necessary to execute them in a controlled and
reproducible environment like AWS EC2. Therefore, the SRK can interact with
containers that run on remote machines via SSH.

To enable the functionality, add the optional ``remote`` section to the
configuration. Additionally the ``address`` value has to be set to the public
IP or domain of the remote server. Note that with a remote configuration the
``lambci`` directory lives on the remote server.

::

	default-provider : "lambci"

	providers :
	  lambci :
	    faas : "lambciLambda"

	service :
	 faas :
	    lambciLambda:
	      # optional remote configuration
	      # if set the directory value below is bound to the specified host
	      remote:
	        # path to scp command
	        scp : '/usr/bin/scp'
	        # path to ssh command
	        ssh : '/usr/bin/ssh'
	        # IP or hostname of server running the lambci/lambda docker image
	        host : 'ec2-instance'
	        # user for scp + ssh
	        user : 'ubuntu'
	        # key file for scp + ssh
	        pem : '~/.aws/AWS.pem'
	      # path to the lambci directory
	      directory : '~/lambci'
	      # address of lambci server API
	      address : 'ec2-instance:9001'
