SRCS = $(shell find . -type f -iname '*.go')

.PHONY: bin env

bin: target/emitio_linux_amd64

target/emitio_linux_amd64: $(SRCS) Makefile
	docker build -t emitio-build .
	docker run \
		--rm \
		-v $(shell pwd):/root/go/src/github.com/supershabam/emitio/emitio \
		-w /root/go/src/github.com/supershabam/emitio/emitio \
		emitio-build \
		/usr/local/go/bin/go build -o $@ cmd/emitio/main.go

pb/emitio/emitio.pb.go: emitio.proto
	protoc emitio.proto --go_out=plugins=grpc:./pb/emitio

target/libtransformd.so_linux_amd64: $(SRCS) Makefile

env:
	docker build -t emitio-build .
	docker run \
		--rm \
		-ti \
		-v $(shell pwd):/root/go/src/github.com/supershabam/emitio/emitio \
		-w /root/go/src/github.com/supershabam/emitio/emitio \
		emitio-build