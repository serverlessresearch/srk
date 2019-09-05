#!/bin/bash

protoc -I pkg/objstore pkg/objstore/objstore.proto --go_out=plugins=grpc:pkg/objstore