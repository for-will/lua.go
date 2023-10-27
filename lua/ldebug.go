package golua

import (
	"log"
)

func (L *LuaState) gRunError(format string, args ...interface{}) {
	// todo: gRunError
	log.Printf(format, args...)
}

func (L *LuaState) gConcatError(p1 StkId, p2 StkId) {
	// todo: gConcatError
	log.Println("concat error")
}

func (L *LuaState) gTypeError(o *TValue, op string) {
	// todo: gTypeError
	log.Println("type error")
}
