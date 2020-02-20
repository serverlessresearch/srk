.. _example_gpu:

===============================================================================
Running GPU-Enabled Functions
===============================================================================
Some FaaS implementations support attached GPUs. This FaaS+GPU approach means
you can run unmodified GPU code (e.g. CUDA) inside your function with a local
GPU. We provide a simple vector addition example for this.

*******************************************************************************
Prerequisites
*******************************************************************************
For this example, we will assume you've already gone through the
:ref:`tutorial_quickstart`. In particular, you will need to use our `fork of
OpenLambda <https://github.com/nathantp/open-lambda>`_ in order to get GPU
spport.

You will also need to have a GPU available on the server running the OpenLambda
worker and all the needed drivers and toolchain for that GPU.

*******************************************************************************
Building and Testing the Example Workload Without FaaS
*******************************************************************************
Before we try to run our GPU-enabled function in FaaS, we should test it
locally to make sure everything works with our system. Navigate to
``examples/gpu`` and then build and test the workload:

::

  $ cd examples/gpu
  $ make
  $ python3 f.py

If all goes well, you should see ``Success`` printed and a clean exit status.
If this doesn't work, you may need to investigate your local GPU environment.


*******************************************************************************
Launching the GPU Workload in FaaS
*******************************************************************************
Now that you know the function works correctly, you can now configure
OpenLambda to run with GPU support. To do this, add ``"enable_gpu" : true`` to
the ``features`` section of our OpenLambda configuration (typically located at
``open-lambda/default-ol/config.json``). It should look like:

::

    "features": {
			"reuse_cgroups": false,
			"import_cache": false,
			"downsize_paused_mem": true,
			"enable_gpu" : true
  	},

You can now configure SRK to use OpenLambda (as described in
:ref:`tutorial_quickstart`), and run a one-shot test of the GPU example:

::

	$ ./srk function create -s examples/gpu
	$ ./srk bench -b one-shot -a '{"test-size":1048576}' -n gpu

Once again, you should see "Success" printed as the response from the function.

You are now ready to start experimenting with GPU-enabled functions. You should
be able to do anything with the GPU in the function that you could do locally.
Keep in mind that OpenLambda limits the available function concurrency to the
number of GPUs when gpu mode is enabled to avoid collisions with devices. This
may limit the number of distinct functions you can have active on a particular
worker. We hope improve this functionality in the future.
