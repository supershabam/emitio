SRCS = $(shell find . -type f -iname '*.go')

.PHONY: bin

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