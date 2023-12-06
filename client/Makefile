# generate *_mock for each *.go file in client/ except *_test.go

ALL_TARGET := $(shell find ./ | grep ".go" | grep -v "_test.go" | grep -v "mock" | sed -e "s/\.go/_mock_test.go/g")

.PHONY: all
all: $(ALL_TARGET)
%_mock_test.go: %.go
	mockgen -source=$< -destination=$@ -package=client
	touch $@