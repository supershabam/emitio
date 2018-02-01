package main

// #cgo LDFLAGS: -L${SRCDIR} -lChakraCore
/*
#include "ChakraCore.h"
#include <stdlib.h>
*/
import "C"
import (
	"fmt"
	"unsafe"
)

var jsInvalidReference C.JsContextRef

func main() {
	var runtime C.JsRuntimeHandle
	var context C.JsContextRef
	var result C.JsValueRef
	var currentSourceContext C.JsSourceContext
	var errCode C.JsErrorCode
	errCode = C.JsCreateRuntime(C.JsRuntimeAttributeNone, nil, &runtime)
	if errCode != C.JsNoError {
		panic("error code")
	}
	errCode = C.JsCreateContext(runtime, &context)
	if errCode != C.JsNoError {
		panic("error code")
	}
	errCode = C.JsSetCurrentContext(context)
	if errCode != C.JsNoError {
		panic("error code")
	}
	var fname C.JsValueRef
	cs := C.CString("sample")
	errCode = C.JsCreateString(cs, C.size_t(len("sample")), &fname)
	if errCode != C.JsNoError {
		panic("error code")
	}
	C.free(unsafe.Pointer(cs))
	var scriptSource C.JsValueRef
	script := `(()=>{return 'Hello World!';})()`
	cscript := C.CString(script)
	errCode = C.JsCreateExternalArrayBuffer(unsafe.Pointer(cscript), C.uint(len(script)), nil, nil, &scriptSource)
	if errCode != C.JsNoError {
		panic("error code")
	}
	// Run the script.
	errCode = C.JsRun(scriptSource, currentSourceContext, fname, C.JsParseScriptAttributeNone, &result)
	if errCode != C.JsNoError {
		panic("error code")
	}
	currentSourceContext++
	// Convert your script result to String in JavaScript; redundant if your script returns a String
	var resultJSString C.JsValueRef
	errCode = C.JsConvertValueToString(result, &resultJSString)
	if errCode != C.JsNoError {
		panic("error code")
	}
	// project result to gostring
	var stringLength C.size_t
	errCode = C.JsCopyString(resultJSString, nil, 0, &stringLength)
	if errCode != C.JsNoError {
		panic("error code")
	}
	fmt.Printf("string_length=%d\n", stringLength)
	resultcs := C.CString("some long thing that hopefully has enough space yo")
	errCode = C.JsCopyString(resultJSString, resultcs, stringLength+1, nil)
	if errCode != C.JsNoError {
		panic("error code")
	}
	fmt.Printf("result=%s", C.GoString(resultcs))
	C.free(unsafe.Pointer(resultcs))
	errCode = C.JsSetCurrentContext(jsInvalidReference)
	if errCode != C.JsNoError {
		panic("error code")
	}
	errCode = C.JsDisposeRuntime(runtime)
	if errCode != C.JsNoError {
		panic("error code")
	}
}
