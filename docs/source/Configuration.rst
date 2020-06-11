.. _configuration:

======================
Configuring SRK
======================

.. _config-install:

******************************
Installation
******************************
SRK relies on a number of runtime files to operate. It also needs somewhere to
store generated intermediate files. The ``./runtime`` directory can be used for
this purpose or you can use the ``install.sh`` script to copy the needed files
to a global location (e.g. ``~/.srk``). Wherever this directory is, you must
inform SRK of it using the SRKHOME environment variable. If this is not set,
SRK will use ./runtime which will only work if you call srk from the
repositories root directory. SRKHOME should be an absolute path.

We recommend adding SRKHOME to your .bashrc (or equivalent) and placing the srk
binary on your PATH (either by moving it somewhere or adding the srk repository
to your PATH).

******************************
Configuration Files
******************************
Configuration of SRK is handled by a configuration file. This file is typically
in yaml format, but SRK can handle the following formats as well: JSON, TOML,
YAML, HCL, INI, envfile and Java Properties files. SRK will look
for a config file at ``$SRKHOME/config.yaml`` (".yaml" can be replaced with any
of the support file format suffixes).

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
