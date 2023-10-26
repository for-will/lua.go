package golua

import "fmt"

func (L *LuaState) DebugRunError(format string, args ...interface{}) {
	// todo: DebugRunError
	fmt.Printf(format, args...)
}

func (L *LuaState) DebugConcatError(p1 StkId, p2 StkId) {
	// todo: DebugConcatError
	fmt.Printf("DebugConcatError")
}
