#!/bin/bash

# golang
protoc -I pkg/objstore pkg/objstore/objstore.proto --go_out=plugins=grpc:pkg/objstore
# c++
protoc -I pkg/objstore --cpp_out=pkg/objstore/ pkg/objstore/objstore.proto
protoc -I pkg/objstore --grpc_out=pkg/objstore/ --plugin=protoc-gen-grpc=`which grpc_cpp_plugin` pkg/objstore/objstore.proto
