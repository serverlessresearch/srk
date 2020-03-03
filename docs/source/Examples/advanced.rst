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
Using Layers
*******************************************************************************

Layers_ provide additional code or files that can be shared between functions.
The FaaS provider hosts a pool of layer packages that can be managed by SRK
using the ``layer`` CLI command and then linked to the function.

The following command packages the files in ``source-dir`` as a layer with name
``layer-name`` and uploads it to the FaaS provider. It also specifies that the
layer is compatible with the runtimes of Golang and Python.

::

	$ ./srk layer create -s <source-dir> -n <layer-name> -r go1.x,python3.8

The command returns the layer identifier that can be used to connect a layer
with a function. The layer files will be available for the function in the
``/opt`` directory. To connect multiple layers the ``--layers`` parameter can
take a comma-separated list of layer identifiers.

::

	$ ./srk function create -s <source-dir> -l <layer-id>


*******************************************************************************
Setting environment variables
*******************************************************************************
Environment_ variables can provide additional information to the function and
the runtime environment. They can be set by adding the ``--env`` parameter at
function creation.

::

	$ ./srk function create -s <source-dir> -e VAR1=VALUE1,VAR2=VALUE2


.. _Layers: https://docs.aws.amazon.com/lambda/latest/dg/configuration-layers.html
.. _Environment: https://docs.aws.amazon.com/lambda/latest/dg/configuration-envvars.html