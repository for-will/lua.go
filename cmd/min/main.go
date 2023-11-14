package main

import (
	"fmt"
	golua "luar/lua"
)

func print(L *golua.LuaState) int {
	var n = L.GetTop()
	for i := 1; i <= n; i++ {
		if i > 1 {
			fmt.Printf("\t")
		}
		if L.IsString(i) {
			fmt.Printf("%s", L.ToString(i))
		} else if L.IsNil(i) {
			fmt.Printf("%s", "nil")
		} else if L.IsBoolean(i) {
			if L.ToBoolean(i) {
				fmt.Printf("%s", "true")
			} else {
				fmt.Printf("%s", "false")
			}
		} else {
			fmt.Printf("%s:%p", L.LTypeName(i), L.ToPointer(i))
		}
	}
	fmt.Printf("\n")
	return 0
}

func main() {
	var L = golua.LuaOpen()
	L.Register("print", print)
	if L.LDoFile("hello.lua") != 0 {
		fmt.Errorf("%s\n", L.ToString(-1))
	}
	L.Close()
}
