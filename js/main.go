package main

import (
	"fmt"

	"github.com/robertkrimen/otto"
)

func main() {
	vm := otto.New()
	_, err := vm.Run(`
		function process(acc, lines) {
			var seq = parseInt(acc, 10)
			var out = []
			for (var i = 0; i < lines.length; i++) {
				var line = lines[i]
				seq = seq + 1
				out.push("" + seq + ":" + line)
			}
			return ["" + seq, out]
		}
	`)
	if err != nil {
		panic(err)
	}
	fn, err := vm.Get("process")
	if err != nil {
		panic(err)
	}
	v, err := fn.Call(fn, "1", []string{"one", "two"})
	if err != nil {
		panic(err)
	}
	fmt.Printf("%+v\n", v)
}
