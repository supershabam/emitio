SRCS = $(shell find . -type f -iname '*.go')

.PHONY: bin generate

bin: target/eio_darwin_amd64

generate:
	protoc -I../edge edge.proto --go_out=plugins=grpc:pb/edge

target/eio_darwin_amd64: $(SRCS) Makefile
	go build -o $@ cmd/eio/main.go

