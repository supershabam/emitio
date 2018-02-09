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

func main() {
	acc := C.CString("acc")
	defer C.free(unsafe.Pointer(acc))
	lines := toCStringsArray([]string{"hi"})
	defer C.freeCharArray(lines, 1)
	var out (*C.char)
	rc := C.transform(acc, lines, 1, &out)
	if rc != 0 {
		panic("error")
	}
	fmt.Println(C.GoString(out))
	C.free(unsafe.Pointer(out))
}
