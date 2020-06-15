======================
Configuring SRK
======================
Configuration of SRK is handled by a configuration file. This file is typically
in yaml format, but SRK can handle the following formats as well: JSON, TOML,
YAML, HCL, INI, envfile and Java Properties files. By default, SRK will look
for a config file at ``configs/srk.yaml``, but you can override this behavior
by passing the ``--config`` option to the CLI (or the ``config-file`` key when
calling srkmgr.NewManager().

*******************************
Configuration Options
*******************************

service
=====================================
This section contains descriptions of individual services, organized by service
category.

faas
----------------------
Function-as-a-Service.

openLambda
^^^^^^^^^^^^^^^^^^^^^^
OpenLambda (OL) is an open source FaaS service that is built on Docker. OL is
relatively easy to get up and running locally or in a small cluster. Because it
is open source, it is a good target for experimenting FaaS features or
modifications.

.. Note:: SRK uses a `fork of OpenLambda
   <https://github.com/NathanTP/open-lambda>`_ to get a few additional features.
   Make sure you use this fork instead of the upstream version (for now).

.. _config-olcmd:

olcmd
"""""""""""""""""""
SRK's OL wrapper needs to know which binary to invoke when interacting with a
local instance. Download and compile OL locally and then set this option to the
path to the ``ol`` binary.

.. _config-oldir:

oldir
""""""""""""""""""""
OL uses a local directory to register functions and provide runtime files. You
can create this directory in OL by running ``./ol new`` (which creates the
directory ``default-ol`` by default). SRK needs to know where this is in order
to register new functions.

awsLambda
^^^^^^^^^^^^^^^
This is the Amazon Web Services implementation of function-as-a-service. This
service will register and launch your functions in the public cloud (with all
associated costs). 

role
""""""""""""""""""""
This is your 'arn' role within Amazon. You can see details in `Amazon's Documentation <https://docs.aws.amazon.com/lambda/latest/dg/lambda-intro-execution-role.html>`_.

vpc-config
"""""""""""""""""""""
If you would like to use a custom vpc for your functions, you can configure
that here. If you don't know what this is, you can leave it as null and SRK
will use Amazon's default behavior.

runtimes
"""""""""""""""""""""
You can set up a list of runtimes for your functions here. A runtime consists
of an AWS provided ``base`` (`<https://docs.aws.amazon.com/lambda/latest/dg/lambda-runtimes.html>`_)
and optional additional ``layers`` (`<https://docs.aws.amazon.com/lambda/latest/dg/configuration-layers.html>`_).
Layers have to be uploaded to AWS lambda beforehand and then specified in the
configuration by their ARN.

::

      # Optional custom runtime and layer configuration
      runtimes :
        # example custom runtime definition
        cffs-python :
          # AWS runtime as base, use 'provided' for custom runtime
          base : provided
          # list of additional layers
          layers :
            # e.g. - 'arn:aws:lambda:eu-central-1:123459789012:layer:runtime-python37:3'

default-runtime
"""""""""""""""""""""
This specifies the runtime to use if it is not given as CLI parameter.

lambciLambda
^^^^^^^^^^^^^^^
`LambCI lambda <https://hub.docker.com/r/lambci/lambda/>`_ provides a
Docker-based sandbox environment that mimics AWS lambda. This can be used to
test AWS lambda functions on a local machine which is easier to debug and does
not create AWS costs. See ``Examples / How to use LambCI lambda`` for a setup
guide.

address
"""""""""""""""""""""
The server address of the Lambci lambda invocation API, usually
``<hostname>:9001``.

directory
"""""""""""""""""""""
The path to the Lambci lambda work directory.

runtimes
"""""""""""""""""""""
As the LambCI lambda runtimes is included in the used Docker image, the runtimes
section does configure additional layers only. Here, layers are directories inside
the layers directory of the Lambci lambda work directory that need to be created
manually beforehand.

::

      # runtime configuration
      runtimes :
        # example runtime definition
        cffs-python :
          # list of directories that make up the runtime
          layers :
            - 'runtime-python37-1'

default-runtime
"""""""""""""""""""""
This specifies the runtime to use if it is not given as CLI parameter.

remote
"""""""""""""""""""""
This section must be configured only if LambCI lambda is running on a remote
machine. It contains the information necessary to execute shell commands via 
SSH.

::

      # optional remote configuration
      # if set the directory value below is bound to the specified host
      remote:
        # path to local scp command if not in path
        scp : '/usr/bin/scp'
        # path to local ssh command if not in path
        ssh : '/usr/bin/ssh'
        # IP or hostname of server running the lambci/lambda docker image
        host : 'ec2-instance'
        # user for scp + ssh
        user : 'ubuntu'
        # key file for scp + ssh
        pem : '~/.aws/AWS.pem'


global
^^^^^^^^^^^^^^^^^^^^
This section provides global behaviors for all FaaS implementations. Note that
some implementations may not support all options. There are currently no global
options.

providers
=======================
A provider aggregates at most one instance of each service category. You can
think of the provider as complete cloud environment. SRK defines two default
providers ('aws' and 'local'), but users are encouraged to implement their own
as needed. A provider entry takes the form:

::

   NAME:
      faas: FAAS_SERVICE
      objStore: OBJ_SERVICE

default-provider
=====================================
Users may define as many providers as they want, but only one provider may be
active at a time. The ``default-provider`` option specifies which provider SRK
should use when running commands.
