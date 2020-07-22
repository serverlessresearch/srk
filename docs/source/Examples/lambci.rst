.. _example_lambci:

===============================================================================
How to use LambCI lambda
===============================================================================

This example shows how to setup the LambCI Lambda Docker container and use it
with the SRK.

*******************************************************************************
Create a configuration file
*******************************************************************************

Before running this example you need to configure SRK with the LambCI provider.
We start out running this example locally, which requires you to specify two
things in the configuration: the data directory and and the address of the 
LambCI server.

For this example, install a configuration file at ``$SRK_HOME/config.yaml`` with the
following content:

::

	default-provider : 'lambci'

	providers :
	  lambci :
	    faas : 'lambciLambda'

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

*******************************************************************************
Install the function
*******************************************************************************

Install the ``echo`` example function using SRK with the following command:

::

	$ ./srk function create -s examples/echo


*******************************************************************************
Start the LambCI server
*******************************************************************************

SRK does not presently start the LambCI FaaS provider on its own, so you will
need to start it manually. The following starts up a container
running a LambCI server locally:

::

	LAMBCI_PATH=$HOME/lambci
	LAMBCI_RUNTIME=python3.8
	LAMBCI_HANDLER=lambda_function.lambda_handler
	docker run --rm -d \
	  --name srk-lambci \
	  -v $LAMBCI_PATH/task:/var/task:ro,delegated \
	  -v $LAMBCI_PATH/runtime:/opt:ro,delegated \
	  --env-file $LAMBCI_PATH/env \
	  -e DOCKER_LAMBDA_STAY_OPEN=1 \
	  -p 9001:9001 \
	  lambci/lambda:$LAMBCI_RUNTIME \
	  $LAMBCI_HANDLER

You may also want to create a script to streamline this process.
See `Scripting LambCI Startup`_.

*******************************************************************************
Invoke the function
*******************************************************************************

Use SRK to invoke the ``echo`` function using the following command:

::

	./srk bench -b one-shot -a '{"hello" : "world"}'


*******************************************************************************
Shut down the LambCI server
*******************************************************************************

::

	docker kill srk-lambci

*******************************************************************************
Using Custom Libraries
*******************************************************************************

SRK with LambCI allows you to add custom libraries to the runtime. The
configuration procedure parallels that of AWS Lambda, which users
`layers <https://docs.aws.amazon.com/lambda/latest/dg/configuration-layers.html>`_.

In this example we will create a runtime that includes Python's popular `Requests <https://requests.readthedocs.io/en/master/>`_.
Use the commands below to install the layer:

::

	mkdir -p $HOME/lambci/layers/requests-layer/python
	pip3 install requests -t $HOME/lambci/layers/requests-layer/python

Now update your configuration file ``$SRKHOME/config.yaml`` as follows:

::

	default-provider : 'lambci'

	providers :
	  lambci :
	    faas : 'lambciLambda'

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
	            - 'requests-layer'

Now recreate the function using the requests layer:

::

	./srk function create -s examples/requests -r with-requests


Run the Docker command (see `Start the LambCI server`_).

Invoke the function

::

	./srk bench -b one-shot -a '{}'


*******************************************************************************
Using a custom runtime
*******************************************************************************

A custom runtime replaces the runtime environment provided by the FaaS provider
with an own runtime package. This package has to be uploaded as a layer to the
FaaS provider.

To use a custom runtime, specify ``provided`` as the runtime name for the
Docker command.

::

	$ ./lambci.sh ~/lambci provided lambda_function.lambda_handler

The lambda container now expects the custom lambda runtime in the ``runtime``
directory. For this, create a layer that contains the runtime code and configure
it in the configuration.

::

	default-provider : 'lambci'

	providers :
	  lambci :
	    faas : 'lambciLambda'

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
	            - 'requests-layer'

The custom runtime can then be specified at function creation. In the example
above, SRK will copy the contents of the ``custom-python`` directory (the
custom runtime) and the ``requests`` layer to the ``runtime`` directory so that
the LambCI ``provided`` container finds it in ``/opt``.

::

	$ ./srk function create -s examples/echo -r custom-runtime


*******************************************************************************
Scripting LambCI Startup
*******************************************************************************

LambCI provides docker images for various runtimes out of the box, but can also
use a custom runtime. To inject the lambda function into the container, the
``/var/task`` and ``/opt`` directories are mounted to local directories by the
``docker run`` command. To be compatible with the SRK these directories need to
be inside the configured LambCI home directory and have the names ``task`` for
the lambda function and ``runtime`` for a custom runtime or additional layer
files.

Instead of running the lambda function immediately, SRK uses the LambCI-provided
webserver with an invocation API to execute the lambda function. Therefore the
port of the webserver has to be exposed by the ``docker run`` command.

It is also possible to inject environment variables via the ``--env-file``
parameter of ``docker run``.

Updates to the function, the runtime files or the environment file require a
restart of the container. The helper program ``entr`` can be used to automate
this. It can be installed via ``apt install entr`` (Ubuntu),
``yum install entr`` (Amazon Linux 2) or ``brew install entr`` on MacOS X.

Please see the following shell script as a loader for a LambCI lambda
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

	default-provider : 'lambci'

	providers :
	  lambci :
	    faas : 'lambciLambda'

	service :
	 faas :
	    lambciLambda:
	      # optional remote configuration
	      # if set the directory value below is bound to the specified host
	      remote:
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

In case the ``ssh`` and ``scp`` commands on your local machine are not in
``$PATH``, the executables can also be set in the remote configuration section:

::

	      ...
	      remote:
	        # path to local scp command
	        scp : '/usr/bin/scp'
	        # path to local ssh command
	        ssh : '/usr/bin/ssh'
	        ...
