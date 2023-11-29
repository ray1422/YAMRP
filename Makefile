# target for gRPC protobuf for golang
PROTOBUF_TARGET := proto/yampr.pb.go proto/yampr_grpc.pb.go
ALL_TARGET := $(PROTOBUF_TARGET)

# make protobuf target
$(PROTOBUF_TARGET): proto/yampr.proto
	protoc --go_out=. --go_opt=paths=source_relative \
    	--go-grpc_out=. --go-grpc_opt=paths=source_relative \
    	proto/yampr.proto

