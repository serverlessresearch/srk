.. _tutorial_quickstart:

======================
Quick Start Tutorial
======================
To get started with SRK, we will test a simple 'echo' cloud function on two
different function-as-a-service platforms. For this tutorial, we will be
focusing on SRK's command line interface that you might use while developing an
application and manually experimenting with services.


************************
Building SRK
************************
SRK is written in the Go language and uses Go's tools to build and test. To get
started with Go, you can with their `documentation
<https://golang.org/doc/install>`_. Note that SRK uses Go's new modules system,
this means that you **do not** need to clone SRK to your $GOHOME in order to use
it. With Go installed, you can build SRK with:

::

   $ go build

Go will automatically download and compile any dependencies and then compile
the SRK CLI ``srk``.


************************
Function Source Code
************************
For this exercise, we will use the provided echo example at ``examples/echo``.
In this directory, you will see three files: 

* **echo.py**: This is our actual function logic, it can be anything you want as
  long as all of it's dependencies are in the echo/ folder that we pass to
  SRK. 
* **f.py**: This is the open-lambda glue code. OL requires that functions be
  named ``f()`` and live in ``f.py``. The signature of ``f()`` must be
  preserved. The body of ``f()`` does any OL-specific actions and calls
  ``echo()`` with the correct arguments.
* **aws.py**: This is the aws Lambda glue code. AWS lambda is more flexible with
  function naming, but has an incompatible signature requirement (it includes a
  'context' field that OL doesn't). SRK requires that you include an ``aws.py``
  with a function ``f(event, context)``.

Since ``f.py`` and ``aws.py`` provide only simple wrappers, we will focus on
``echo.py``. This file provides our cloud function ``echo()`` which simply
returns any request it receives.

*************************
SRK Configuration
*************************
You can see full details at :ref:`configuration`.

Initial Setup (Installation)
=================================
For this tutorial, we will use the builtin runtime directory as our SRK home.
This directory has everything SRK will need in order to operate at runtime. You
may also use the ./install.sh script to place this somewhere else (see
:ref:`config-install` for more details). For now, let's export the SRKHOME variable so srk knows where to find its files:

::

   $ export SRKHOME=/PATH/TO/SRKREPO/runtime

Replace /PATH/TO/SRKREPO with wherever you repository is. SRKHOME should be an
abosulte path.

Configuration File
=================================
SRK provides an interchangable interface to multiple providers of standard
cloud services. In this case, we will focus on two different FaaS providers:
AWS Lambda and OpenLambda. At the moment, SRK does not manage the configuration
of these providers, so we'll need to do that now. First, let's create a default
configuration from the template provided with SRK:

::

   $ cp runtime/example-config.yaml runtime/config.yaml

In the next two subsections, we will customize this config to include
OpenLambda and AWS Lambda support.

OpenLambda
^^^^^^^^^^^^^^^^^^^^^
OpenLambda is an open-source function-as-a-service provider based linux
containers. We maintain a fork of this project to provide additional features.
Go ahead and clone this repo anywhere in your filesystem, we'll use our home
directory in these instructions:

::

   $ cd ~
   $ git clone git@github.com:NathanTP/open-lambda.git

Next, follow the `instructions in open-lambda's README
<https://github.com/NathanTP/open-lambda/blob/master/README.md>`_ to compile and
install the system. You will need to have Docker and Golang installed before
starting this step.

With OpenLambda built, we can now configure SRK to use it. Open up
``runtime/config.yaml`` and modify the ``service.faas.openLambda`` section to look
as follows:

::

   faas:
      openLambda:
         olcmd: "~/open-lambda/ol"
         oldir: "~/open-lambda/default-ol"

``olcmd`` should point to the OpenLambda binary you produced when building the
project. ``oldir`` should point to an OpenLambda workspace (created by calling
``./ol new``). See the :ref:`config-olcmd` and :ref:`config-oldir` sections for
more details.

AWS Lambda
^^^^^^^^^^^^^^^^^^^^^
A full guide to using AWS Lambda is beyond the scope of this document, but you
can follow AWS's tutorial `here
<https://docs.aws.amazon.com/lambda/latest/dg/getting-started.html>`_. For SRK
to work, you will need to provide an ARN role and optional VPC. You can set
these values in ``runtime/config.yaml`` in the ``service.faas.awsLambda`` section.
For example:

::

   faas:
      awsLambda:
         role: "arn:aws:iam::123459789012:role/service-role/my-service-role-ae04d032"
         vpc: null

The role is an AWS-specific set of permissions for your function. You can learn
more about creating roles `here
<https://docs.aws.amazon.com/lambda/latest/dg/lambda-intro-execution-role.html>`_.
The vpc setting controls networking for your function, unless you have a
specific use-case, you can leave this as null (for more information, see the `AWS VPC
documentation
<https://docs.aws.amazon.com/lambda/latest/dg/configuration-vpc.html>`_).

Setting the current provider
^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^
Now that we have both AWS Lambda and OpenLambda configured, we can switch
between them by changing the ``default-provider`` option in
``runtime/config.yaml``. To start with, let's switch to use local resources only (e.g. OpenLambda):

::

   default-provider: local

To switch to AWS, you can instead set it to ``default-provider: aws``.

************************
Function Installation
************************
With our service providers configured, we can now proceed to packaging and
installing our function:

::

   $ ./srk function create --source examples/echo

This command packaged up our source code in a format compatible with OpenLambda
and installed it to the directory we configured earlier. To install to AWS,
change your ``default-provider`` in ``runtime/config.yaml`` to AWS and re-run the
same command. In this case, SRK created a zip file and uploaded it to Amazon's
service using their Golang bindings.

************************
Function Invocation
************************
SRK provides simple benchmarks that you can run from the command line to
interact with your newly created functions. In this example, we will use the
'one-shot' benchmark that synchronously invokes the function exactly once and
displays the response:

::

   $ ./srk bench --bench one-shot --function-args '{"hello" : "world"}' --function-name echo

You should see {"hello" : "world"} printed on your screen. Try passing
different arguments, your function should simply return whatever you pass it.

This benchmark ran against AWS Lambda, to try OpenLambda, switch your
``runtime/config.yaml`` back to using local resources and repeat the command.

*******************
Next Steps
*******************
You may new begin experimenting with different functions. Make some
modifications to ``echo.py`` or write your own new function. You will need to
run ``./srk create ...`` again to upload the new function. Once you are
comfortable with the behavior of your function, head over to our `GoDoc Pages
<https://godoc.org/github.com/serverlessresearch/srk/pkg/srkmgr>`_ to learn
how to write more advanced benchmarks using the programmatic interface to SRK.
