PROTO=srkproto/srkproto.pb.go 
SERVER=server/server

all: $(PROTO) $(CLIENT) $(SERVER)

.PHONY: proto
proto: $(PROTO)

$(PROTO): srkproto/srkproto.proto
	 protoc -I srkproto/ srkproto/srkproto.proto --go_out=plugins=grpc:srkproto

.PHONY: server
server: $(SERVER)

$(SERVER): server/server.go $(PROTO)
	cd server && go build server.go

clean:
	rm -f server/server
	rm -f client/client
	rm -f srkproto/srkproto.pb.go
