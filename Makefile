.PHONY: all 
# target for gRPC protobuf for golang
ALL_TARGET := proto client
all: client proto

client: $(wildcard ./client/*.go)
	make -C client

proto: $(wildcard ./proto/*.proto)
	make -C proto

# make protobuf target


.PHONY: client proto