SRCS = $(shell find . -type f -iname '*.go')

.PHONY: bin

bin: target/emitio_linux_amd64

target/emitio_linux_amd64: $(SRCS) Makefile
	docker run \
		--rm \
		-v $(shell pwd):/go/src/github.com/supershabam/emitio/emitio \
		-w /go/src/github.com/supershabam/emitio/emitio \
		golang:1.9.2-stretch \
		go build -o $@ cmd/emitio/main.go

pb/emitio/emitio.pb.go: emitio.proto
	protoc emitio.proto --go_out=plugins=grpc:./pb/emitio