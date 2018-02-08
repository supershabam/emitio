package transformers

// #cgo LDFLAGS: -L${SRCDIR} -lChakraCore
/*
#include "ChakraCore.h"
#include <stdlib.h>
*/
import "C"
import (
	"context"
	"fmt"
	"runtime"
	"unsafe"

	"github.com/pkg/errors"
)

var jsInvalidReference C.JsContextRef

// chakra core makes use of thread-local storage, so we must make sure that
// the goroutine interacting with its code is the same os thread.
var workCh chan func()

func init() {
	workCh = make(chan func())
	go func() {
		runtime.LockOSThread()
		defer runtime.UnlockOSThread()
		for work := range workCh {
			work()
		}
	}()
}

func do(fn func() error) error {
	var err error
	done := make(chan struct{})
	workCh <- func() {
		defer close(done)
		err = fn()
	}
	<-done
	return err
}

type JS struct {
	runtime *C.JsRuntimeHandle
	context *C.JsContextRef
}

func addref(r C.JsRef, name string) {
	var errCode C.JsErrorCode
	var count C.uint
	errCode = C.JsAddRef(r, &count)
	if errCode != C.JsNoError {
		panic("add ref for " + name)
	}
	// zap.L().Info("added ref", zap.String("name", name), zap.Int("count", int(count)))
}

func rmref(r C.JsRef, name string) {
	var errCode C.JsErrorCode
	var count C.uint
	errCode = C.JsRelease(r, &count)
	if errCode != C.JsNoError {
		panic("rm ref for " + name)
	}
	// zap.L().Info("removed ref", zap.String("name", name), zap.Int("count", int(count)))
}

func javascriptify(value interface{}) (*C.JsValueRef, error) {
	var result C.JsValueRef
	var errCode C.JsErrorCode
	switch value := value.(type) {
	case []string:
		errCode = C.JsCreateArray(C.uint(len(value)), &result)
		if errCode != C.JsNoError {
			return nil, errors.New("js create array")
		}
		for i, v := range value {
			val, err := javascriptify(v)
			if err != nil {
				return nil, err
			}
			var idx C.JsValueRef
			errCode = C.JsDoubleToNumber(C.double(float64(i)), &idx)
			if errCode != C.JsNoError {
				return nil, errors.New("js double to number")
			}
			errCode = C.JsSetIndexedProperty(result, idx, *val)
			if errCode != C.JsNoError {
				return nil, errors.New("js set indexed property")
			}
		}
		return &result, nil
	case string:
		cs := C.CString(value)
		defer C.free(unsafe.Pointer(cs))
		errCode = C.JsCreateString(cs, C.size_t(len(value)), &result)
		if errCode != C.JsNoError {
			return nil, errors.New("js create string")
		}
		return &result, nil
	default:
		return nil, fmt.Errorf("unable to javascriptify value of type %T", value)
	}
}

func golangify(value C.JsValueRef) (interface{}, error) {
	var errCode C.JsErrorCode
	var t C.JsValueType
	errCode = C.JsGetValueType(value, &t)
	if errCode != C.JsNoError {
		return nil, fmt.Errorf("get value type")
	}
	switch t {
	case C.JsArray:
		return golangifyArray(value)
	case C.JsString:
		return golangifyString(value)
	default:
		return nil, fmt.Errorf("unhandled js value type %+v", t)
	}
}

func golangifyString(value C.JsValueRef) (string, error) {
	var errCode C.JsErrorCode
	var l C.size_t
	errCode = C.JsCopyString(value, nil, 0, &l)
	if errCode != C.JsNoError {
		return "", errors.New("copy string get length")
	}
	// cs := C.malloc(C.size_t(C.sizeof(*C.char)) * l)
	buf := make([]byte, l)
	cs := C.CString(string(buf))
	defer C.free(unsafe.Pointer(cs))
	errCode = C.JsCopyString(value, cs, l, nil)
	if errCode != C.JsNoError {
		return "", errors.New("copy string get length")
	}
	return C.GoStringN(cs, C.int(l)), nil
}

func golangifyArray(value C.JsValueRef) ([]interface{}, error) {
	var errCode C.JsErrorCode
	l, err := property(value, "length")
	if err != nil {
		return nil, errors.Wrap(err, "property length")
	}
	var f C.double
	errCode = C.JsNumberToDouble(l, &f)
	if errCode != C.JsNoError {
		return nil, errors.Wrap(err, "number to double")
	}
	result := []interface{}{}
	length := int(f)
	for i := 0; i < length; i++ {
		var idx C.JsValueRef
		errCode = C.JsDoubleToNumber(C.double(float64(i)), &idx)
		if errCode != C.JsNoError {
			return nil, errors.Wrap(err, "double to number")
		}
		var item C.JsValueRef
		errCode = C.JsGetIndexedProperty(value, idx, &item)
		if errCode != C.JsNoError {
			return nil, errors.Wrap(err, "get indexed property")
		}
		gitem, err := golangify(item)
		if err != nil {
			return nil, err
		}
		result = append(result, gitem)
	}
	return result, nil
}

func propertyID(name string) (C.JsPropertyIdRef, error) {
	var result C.JsPropertyIdRef
	var errCode C.JsErrorCode
	cs := C.CString(name)
	defer C.free(unsafe.Pointer(cs))
	errCode = C.JsCreatePropertyId(cs, C.size_t(len(name)), &result)
	if errCode != C.JsNoError {
		return nil, fmt.Errorf("error %+v creating property id=%s", errCode, name)
	}
	return result, nil
}

func property(object C.JsValueRef, name string) (C.JsValueRef, error) {
	var result C.JsValueRef
	var errCode C.JsErrorCode
	propID, err := propertyID(name)
	if err != nil {
		return nil, errors.Wrap(err, "propertyID")
	}
	errCode = C.JsGetProperty(object, propID, &result)
	if errCode != C.JsNoError {
		return nil, fmt.Errorf("get property %s", name)
	}
	return result, nil
}

func (js *JS) Transform(ctx context.Context, acc string, lines []string) (string, []string, error) {
	var v interface{}
	err := do(func() error {
		var errCode C.JsErrorCode
		var (
			global,
			undefined,
			result C.JsValueRef
		)
		errCode = C.JsSetCurrentContext(*js.context)
		if errCode != C.JsNoError {
			return fmt.Errorf("js set current context")
		}
		errCode = C.JsGetGlobalObject(&global)
		if errCode != C.JsNoError {
			return fmt.Errorf("get global object")
		}
		fn, err := property(global, "transform")
		if err != nil {
			return errors.Wrap(err, "property")
		}
		jsAcc, err := javascriptify(acc)
		if err != nil {
			return err
		}
		addref(C.JsRef(*jsAcc), "jsacc")
		defer rmref(C.JsRef(*jsAcc), "jsacc")
		jsLines, err := javascriptify(lines)
		if err != nil {
			return err
		}
		addref(C.JsRef(*jsLines), "jslines")
		defer rmref(C.JsRef(*jsLines), "jslines")
		errCode = C.JsGetUndefinedValue(&undefined)
		if errCode != C.JsNoError {
			return fmt.Errorf("get undefined value")
		}
		errCode = C.JsGetUndefinedValue(&result)
		if errCode != C.JsNoError {
			return fmt.Errorf("get undefined value into result")
		}
		addref(C.JsRef(result), "result")
		defer rmref(C.JsRef(result), "result")
		args := []C.JsValueRef{
			undefined,
			*jsAcc,
			*jsLines,
		}
		// note that args[0] is thisArg of the call; actual args start at index 1
		errCode = C.JsCallFunction(fn, &args[0], C.ushort(len(args)), &result)
		if errCode != C.JsNoError {
			return fmt.Errorf("call function")
		}
		v, err = golangify(result)
		if err != nil {
			return errors.Wrap(err, "golangify")
		}
		return nil
	})
	if err != nil {
		return "", nil, err
	}
	l, ok := v.([]interface{})
	if !ok {
		return "", nil, errors.New("expected transform result to be list")
	}
	if len(l) < 2 {
		return "", nil, errors.New("expected transform to return 2 results")
	}
	acc, ok = l[0].(string)
	if !ok {
		return "", nil, errors.New("expected first result to be string")
	}
	l, ok = l[1].([]interface{})
	if !ok {
		return "", nil, errors.New("expected second result to be list")
	}
	result := []string{}
	for _, li := range l {
		s, ok := li.(string)
		if !ok {
			return "", nil, errors.New("expected second result item to be string")
		}
		result = append(result, s)
	}
	return acc, result, nil
}

func NewJS(script string) (*JS, error) {
	js := &JS{}
	err := do(func() error {
		var (
			runtime C.JsRuntimeHandle
			context C.JsContextRef
		)
		js.runtime = &runtime
		js.context = &context
		var errCode C.JsErrorCode
		errCode = C.JsCreateRuntime(C.JsRuntimeAttributeNone, nil, js.runtime)
		if errCode != C.JsNoError {
			return fmt.Errorf("js create runtime")
		}
		errCode = C.JsCreateContext(*js.runtime, js.context)
		if errCode != C.JsNoError {
			return fmt.Errorf("js create context")
		}
		errCode = C.JsSetCurrentContext(*js.context)
		if errCode != C.JsNoError {
			return fmt.Errorf("js set current context")
		}
		var fname C.JsValueRef
		cs := C.CString("code_source_perhaps_unnecessary")
		defer C.free(unsafe.Pointer(cs))
		errCode = C.JsCreateString(cs, C.size_t(len("code_source_perhaps_unnecessary")), &fname)
		if errCode != C.JsNoError {
			return fmt.Errorf("js create string for source")
		}
		var scriptSource C.JsValueRef
		cscript := C.CString(script)
		defer C.free(unsafe.Pointer(cscript))
		errCode = C.JsCreateExternalArrayBuffer(unsafe.Pointer(cscript), C.uint(len(script)), nil, nil, &scriptSource)
		if errCode != C.JsNoError {
			return fmt.Errorf("create external array buffer")
		}
		// Run the script.
		var currentSourceContext C.JsSourceContext
		errCode = C.JsRun(scriptSource, currentSourceContext, fname, C.JsParseScriptAttributeNone, nil)
		if errCode != C.JsNoError {
			return fmt.Errorf("js run")
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return js, nil
}
