package transformers

// #cgo CFLAGS: -I. -Os -std=c99 -Wall -fstrict-aliasing
// #cgo LDFLAGS: -lm -ltransformd
// #include <stdio.h>
// #include <stdlib.h>
// #include "duktape.h"
// #include "transform.h"
/*

static cstring* makeCharArray(int size) {
        return calloc(sizeof(char*), size);
}

static void setArrayString(cstring* a, char *s, int n) {
        a[n] = s;
}

static char* getArrayString(cstring* a, int n) {
	return a[n];
}

static void freeCharArray(cstring* a, int size) {
        int i;
        for (i = 0; i < size; i++)
                free(a[i]);
        free(a);
}
*/
import "C"
import (
	"context"
	"fmt"
	"unsafe"
)

func toCStringsArray(in []string) *C.cstring {
	a := C.makeCharArray(C.int(len(in)))
	for idx, v := range in {
		C.setArrayString(a, C.CString(v), C.int(idx))
	}
	return a
}

type TransformIn struct {
	acc   string
	lines []string
	in    *C.transform_in
}

func (ti *TransformIn) From(in *C.transform_in) {
	if ti.in != nil {
		ti.Free()
	}
	ti.in = in
	ti.acc = C.GoString(in.accumulator)
	ti.lines = make([]string, in.nlines)
	for i := 0; i < len(ti.lines); i++ {
		ti.lines[i] = C.GoString(C.getArrayString(in.lines, C.int(i)))
	}
}

func (ti *TransformIn) C() C.transform_in {
	if ti.in != nil {
		return *ti.in
	}
	ti.in = new(C.transform_in)
	ti.in.accumulator = C.CString(ti.acc)
	ti.in.lines = toCStringsArray(ti.lines)
	ti.in.nlines = C.int(len(ti.lines))
	return *ti.in
}

func (ti *TransformIn) Free() {
	if ti.in == nil {
		return
	}
	C.freeCharArray(ti.in.lines, ti.in.nlines)
	C.free(unsafe.Pointer(ti.in.accumulator))
	ti.in = nil
}

type JS struct {
	dctx *C.duk_context
}

func NewJS(script string) (*JS, error) {
	var dctx *C.duk_context
	dctx = C.duk_create_heap(nil, nil, nil, nil, nil)
	src := C.CString(script)
	defer C.free(unsafe.Pointer(src))
	C.duk_eval_raw(dctx, src, 0, 0|C.DUK_COMPILE_EVAL|C.DUK_COMPILE_NOSOURCE|C.DUK_COMPILE_STRLEN|C.DUK_COMPILE_NOFILENAME)
	return &JS{
		dctx: dctx,
	}, nil
}

func (js *JS) Transform(ctx context.Context, acc string, lines []string) (string, []string, error) {
	var out C.transform_in
	ti := &TransformIn{
		acc:   acc,
		lines: lines,
	}
	defer ti.Free()
	rc := C.transform(js.dctx, ti.C(), &out)
	if rc != 0 {
		return "", nil, fmt.Errorf("error %d from transform", rc)
	}
	ti.From(&out)
	return ti.acc, ti.lines, nil
}
