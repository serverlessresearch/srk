.. _example_advanced:

===============================================================================
Advanced Function Setup
===============================================================================

This example shows how to setup a function using environment variables and
layers.

*******************************************************************************
Adding Additional Files
*******************************************************************************

Sometimes it is necessary to add additional files from outside the source
directory to the function package. This can be done via the ``--files``
parameter.

::

	$ ./srk function create -s <source-dir> -f <path-to-file1>,<path-to-file2>


*******************************************************************************
Setting Environment Variables
*******************************************************************************
Environment_ variables can provide additional information to the function and
the runtime environment. They can be set by adding the ``--env`` parameter at
function creation.

::

	$ ./srk function create -s <source-dir> -e VAR1=VALUE1,VAR2=VALUE2


*******************************************************************************
Setting the Runtime
*******************************************************************************
If a function does not use the configured default runtime it can be specified
via the ``--runtime`` parameter.

::

	$ ./srk function create -s <source-dir> -r <runtime>

The runtime_ can either be provided by the FaaS provider or defined as a set of
layers_ configured in the configuration.

.. _Runtime: https://docs.aws.amazon.com/lambda/latest/dg/lambda-runtimes.html
.. _Layers: https://docs.aws.amazon.com/lambda/latest/dg/configuration-layers.html
.. _Environment: https://docs.aws.amazon.com/lambda/latest/dg/configuration-envvars.html
