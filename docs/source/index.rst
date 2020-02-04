The Serverless Research Kit (SRK)
=======================================
The Serverless Research Kit (SRK) exists to accelerate research progress in serverless computing.
It aims to make it easy to innovate on services typically offered by cloud providers by including, for example,
readily hackable versions of cloud functions or cloud object storage. SRK also plans to include common benchmarks
and operational tools, so that launching and evaluating a multi-tenant serverless service is quick and easy.

You can think of SRK as taking the role of a cloud provider by providing common
services through a standard API. Unlike a real cloud provider, SRK is designed
to be hackable and pluggable; the same application can run against many
implementations of services. In SRK, we use the term **service** to describe a
particular API (e.g. object storage or function-as-a-service) and **provider**
to mean a coherent collection of services (e.g. the *AWS* provider would use
Lambda for FaaS and S3 for object storage while the *local* provider might use
OpenLambda for FaaS and your local filesystem to provide object storage).

.. Note:: This documentation focuses on general usage and configuration for SRK. For detailed API documentation, check out our `GoDoc Pages <https://godoc.org/github.com/serverlessresearch/srk>`_.

.. toctree::
   :maxdepth: 2
   :caption: Contents:

   QuickStart
   Configuration

.. Indices and tables
.. ==================
..
.. * :ref:`genindex`
.. * :ref:`modindex`
.. * :ref:`search`
