/*

The SRK Object Storage API defines a simple and standardized way of interacting with immutable cloud object storage such as
provided by AWS S3, Google Cloud Storage, and Azure Cloud Storage.
The initial specification provides only minimal functionality, which supports the aim of making new implementations easy to build.
We imagine that the specification may grow slowly, or better yet, support extensions to enable additional functionality.

Limitations and Design Considerations

Access control - this is a critical feature, especially because SRK aims to support multi-tenant settings, but we
have deferred work on this until we devise a general approach. We prefer to start with no access control rather than
add something not and change it later.

Multipart uploads - these are not supported in the current API
The main reason to do this is probably to achieve greater performance through parallel uploads of multiple parts of a
large file. Before adding this complexity we should determine whether it is needed for gRPC, as well as whether gRPC
streams can provide the support needed.

Errors - the standard gRPC status codes (https://github.com/grpc/grpc/blob/master/doc/statuscodes.md) correspond
closely to the needs of object storage, so we encode errors as part of the RPC mechanism rather than in the response
messages. This simplifies error handling because the client only needs one check.

Consistency guarantees - consistency guarantees are left unspecified to allow implementations to experiment with them.
We realize that this may undermine the ability to run multiple applications against the same API, so it may make sense
to incorporate a consistency level into requests.

Object versions - object versions are not part of the initial specification. Versions are related to consistency
guarantees, and may not always be needed. It may make most sense to support versions as an extension.

Physical locations - there is no way to specify that objects should live in a particular geography or data center.
This is an area for future enhancement or extension.

Asynchronous requests - some gRPC clients support asynchronous requests, and we rely on this mechanism rather than
exposing asynchronous requests as part of the API specification.
*/
package objstore