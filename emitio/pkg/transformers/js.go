package transformers

import (
	"context"
	"fmt"

	"github.com/pkg/errors"

	"github.com/robertkrimen/otto"
)

const (
	transformFn = "transform"
)

// JS is a transformer that operates via a javascript runtime
type JS struct {
	vm *otto.Otto
}

// NewJS initializes a javascript runtime to process the provided script
func NewJS(script string) (*JS, error) {
	vm := otto.New()
	_, err := vm.Run(script)
	if err != nil {
		return nil, errors.Wrap(err, "vm run")
	}
	_, err = vm.Get(transformFn)
	if err != nil {
		return nil, errors.Wrap(err, "getting transform function")
	}
	return &JS{
		vm: vm,
	}, nil
}

// Transform implements emitio.Transformer
func (js *JS) Transform(ctx context.Context, acc string, in []string) (string, []string, error) {
	v, err := js.vm.Get("transform")
	if err != nil {
		return "", nil, errors.Wrap(err, "get transform")
	}
	rv, err := v.Call(v, acc, in)
	if err != nil {
		return "", nil, err
	}
	result, err := rv.Export()
	if err != nil {
		return "", nil, err
	}
	r, ok := result.([]interface{})
	if !ok {
		return "", nil, fmt.Errorf("expected js result to be an array but it is %T", result)
	}
	if len(r) < 2 {
		return "", nil, errors.New("expected js result to be an array of at least len 2")
	}
	acci := r[0]
	outi := r[1]
	acc, ok = acci.(string)
	if !ok {
		return "", nil, errors.New("expected js result first element to be string")
	}
	outii, ok := outi.([]string)
	if !ok {
		if li, ok := outi.([]interface{}); ok && len(li) == 0 {
			return acc, []string{}, nil
		}
		return "", nil, fmt.Errorf("expected js result second element to be array of string but it is %T", outi)
	}
	return acc, outii, nil
}
