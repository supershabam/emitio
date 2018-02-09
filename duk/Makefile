#
#  Example Makefile for building a program with embedded Duktape.
#
#  There are two source sets in the distribution: (1) combined sources where
#  you only need duktape.c, duktape.h, and duk_config.h, and (2) separate
#  sources where you have a bunch of source and header files.  Whichever
#  you use, simply include the relevant sources into your C project.  This
#  Makefile uses the combined source file.
#

DUKTAPE_SOURCES = duktape-2.2.0/src/duktape.c duktape-2.2.0/extras/print-alert/duk_print_alert.c

# Compiler options are quite flexible.  GCC versions have a significant impact
# on the size of -Os code, e.g. gcc-4.6 is much worse than gcc-4.5.

CC	= gcc
CCOPTS	= -Os -pedantic -std=c99 -Wall -fstrict-aliasing -fomit-frame-pointer
CCOPTS += -I./duktape-2.2.0/src -I./duktape-2.2.0/extras/print-alert  # for combined sources
CCLIBS	= -lm
DEFINES =

# If you want a 32-bit build on a 64-bit host
#CCOPTS += -m32

# Use the tools/configure.py utility to modify Duktape default configuration:
# http://duktape.org/guide.html#compiling
# http://wiki.duktape.org/Configuring.html

# For debugging, use -O0 -g -ggdb, and don't add -fomit-frame-pointer

transform:	$(DUKTAPE_SOURCES) transform.c
	$(CC) -o $@ $(DEFINES) $(CCOPTS) $(DUKTAPE_SOURCES) transform.c $(CCLIBS)

transform.so:	$(DUKTAPE_SOURCES) transform.c
	$(CC) -shared -fPIC -o $@ $(DEFINES) $(CCOPTS) $(DUKTAPE_SOURCES) transform.c $(CCLIBS)

.PHONY: install
install: transform.so
	cp transform.so /usr/local/lib/libtransform.so