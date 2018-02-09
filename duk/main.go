package main

// #cgo CFLAGS: -I./duktape-2.2.0/src -Os -std=c99 -Wall -fstrict-aliasing -fomit-frame-pointer
// #cgo LDFLAGS: -lduktape -lm -ltransform
// #include <stdio.h>
// #include <stdlib.h>
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

func main() {
	var out C.transform_in
	ti := &TransformIn{
		acc:   "{}",
		lines: []string{"hello", "there"},
	}
	defer ti.Free()
	rc := C.transform(ti.C(), &out)
	if rc != 0 {
		panic("error")
	}
	ti.From(&out)
	fmt.Printf("%+v\n", ti)
	fmt.Printf("nlines=%d\n", out.nlines)
	fmt.Println(C.GoString(out.accumulator))
	C.free(unsafe.Pointer(out.accumulator))
}
