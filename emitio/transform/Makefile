SRCS = $(shell find . -type f -iname '*.c')
OS = unknown
UNAME := $(shell uname)
ifeq ($(UNAME), Linux)
	OS = linux
endif

.PHONY: all clean install

all: target/libtransformd_$(OS)_amd64.so

clean:
	rm -fr target/*

install: target/libtransformd_$(OS)_amd64.so
	cp target/libtransformd_$(OS)_amd64.so /usr/local/lib/libtransformd.so

target/libtransformd_$(OS)_amd64.so: $(SRCS)
	gcc -shared -fPIC -g -o $@ -I./duktape-2.2.0/src duktape-2.2.0/src/duktape.c transform.c -lm

