SRCS = $(shell find . -type f -iname '*.go')

.PHONY: bin generate

bin: target/edge_linux_amd64

generate:
	protoc -I../emitio emitio.proto --go_out=plugins=grpc:pb/emitio
	protoc -I./ edge.proto --go_out=plugins=grpc:pb/edge

target/edge_linux_amd64: $(SRCS) Makefile
	docker run \
		--rm \
		-v $(shell pwd):/go/src/github.com/supershabam/emitio/edge \
		-w /go/src/github.com/supershabam/emitio/edge \
		golang:1.9.2-stretch \
		go build -o $@ cmd/edge/main.go
