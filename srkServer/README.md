# SRK Web Server
This web server acts as a proxy for SRK functionality. This allows SRK to run
on a remote machine. It also allows clients of any language that supports gRPC
to interact with SRK. 

## SRK Server API
The server API is largely a clone of the srk package and should have
most of the same behaviors. Unlike the native library behavior, the srkmgr
package is not exported, instead the server will automatically create a manager
and a session from default behaviors (as specified in srk.yaml) and the same
manager will persist for the lifetime of the server. Currently, there is no way
to reset the manager without restarting the server.

