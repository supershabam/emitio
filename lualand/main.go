package main

import (
	"fmt"
	"unsafe"
)

/*
#cgo CFLAGS: -I/usr/local/include/luajit-2.0 -I/usr/local/musl/include/
#cgo LDFLAGS: -L/usr/local/lib -lluajit-5.1
#include <lua.h>
#include <lualib.h>
#include <lauxlib.h>
#include <stdlib.h>
*/
import "C"

/*
lua reference http://pgl.yoyo.org/luai/i/lua_pushlstring
*/

func main() {
	L := NewState()
	defer L.Close()
	L.OpenLibs()
	C.lua_createtable(L.state, 0, 0)
	t := map[string]string{
		"hello": "there",
		"my":    "goodbuddy",
	}
	for k, v := range t {
		cs := C.CString(v)
		C.lua_pushstring(L.state, cs)
		C.free(unsafe.Pointer(cs))
		cs = C.CString(k)
		C.lua_setfield(L.state, -2, cs)
		C.free(unsafe.Pointer(cs))
	}
	str := "a"
	cs := C.CString(str)
	C.lua_setfield(L.state, C.LUA_GLOBALSINDEX, cs)
	C.free(unsafe.Pointer(cs))
	L.LoadString(`
	for key,value in pairs(a) do 
		print(key,value) 
	end
`)
	L.Run()

	// 	L.LoadString(`
	// local obs_
	// function _G.a_x()
	// 	obs_ = "some text value"
	// end

	// function _G.a_get_obs()
	// 	return obs_
	// end
	// `)
	// cs := C.CString("say")
	// defer C.free(unsafe.Pointer(cs))
	// C.lua_setfield(L.state, C.LUA_GLOBALSINDEX, cs)
	// time.AfterFunc(time.Second, func() {
	// 	L.Close()
	// })
	// str := "hi"
	// cs := C.CString(str)
	// C.lua_pushlstring(L.state, cs, C.size_t(len(str)))
	// C.free(unsafe.Pointer(cs))
	// 	L.LoadString(`
	// print("this is outside my function")
	// return function(input)
	// 	print(input)
	// end
	// `)
	// 	str := "my_function"
	// 	cs := C.CString(str)
	// 	C.lua_setfield(L.state, C.LUA_GLOBALSINDEX, cs)
	// 	C.free(unsafe.Pointer(cs))
	// L.LoadString(`	print("greetings ")`)
	// arg1 := "from go"
	// cs := C.CString(arg1)
	// C.lua_pushstring(L.state, cs)
	// C.free(unsafe.Pointer(cs))
	// if C.lua_pcall(L.state, 1, 0, 0) != 0 {
	// 	panic(L.errorString())
	// }
	// L.Run()
}

// LuaStatePtr is a type to respresent `struct lua_State`
type LuaStatePtr *C.struct_lua_State

// State stores lua state
type State struct {
	state LuaStatePtr
}

// NewState creates new Lua state
func NewState() *State {
	l := C.luaL_newstate()
	return &State{l}
}

// OpenLibs loads lua libraries
func (L *State) OpenLibs() {
	C.luaL_openlibs(L.state)
}

// LoadString loads code into lua stack
func (L *State) LoadString(str string) error {
	csrc := C.CString(str)
	defer C.free(unsafe.Pointer(csrc))

	if C.luaL_loadstring(L.state, csrc) != 0 {
		return fmt.Errorf(L.errorString())
	}

	return nil
}

// Run executes code in stack
func (L *State) Run() error {
	if C.lua_pcall(L.state, 0, 0, 0) != 0 {
		return fmt.Errorf(L.errorString())
	}

	return nil
}

// Close destroys lua state
func (L *State) Close() {
	C.lua_close(L.state)
}

func (L *State) errorString() string {
	return C.GoString(C.lua_tolstring(L.state, -1, nil))
}
