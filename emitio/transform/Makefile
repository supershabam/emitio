SRCS = $(shell find . -type f -iname '*.c')

.PHONY: all clean install

all: target/libtransformd.so

clean:
	rm -fr target/*

install: target/libtransformd.so
	cp target/libtransformd.so /usr/local/lib/

target/libtransformd.so: $(SRCS)
	gcc -shared -fPIC -g -o $@ -I./duktape-2.2.0/src duktape-2.2.0/src/duktape.c transform.c -lm

