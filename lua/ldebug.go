package golua

import "fmt"

func (L *LuaState) DebugRunError(format string, args ...interface{}) {
	// todo
	fmt.Printf(format, args...)
}
