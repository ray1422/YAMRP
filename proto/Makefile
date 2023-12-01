PROTOBUF_TARGET := yampr.pb.go yampr_grpc.pb.go

all: $(PROTOBUF_TARGET)

$(PROTOBUF_TARGET): yampr.proto
	protoc --go_out=. --go_opt=paths=source_relative \
    	--go-grpc_out=. --go-grpc_opt=paths=source_relative \
    	yampr.proto