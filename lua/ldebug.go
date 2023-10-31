package golua

import (
	"log"
)

// ResetHookCount
// 对应C函数：`resethookcount(L)'
func ResetHookCount(L *LuaState) {
	L.hookCount = L.baseHootCount
}

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

func gCheckCode(pt *Proto) int {
	// todo: gCheckCode
	log.Println("gCheckCode not implemented")
	return 0
}

// 对应C函数：int luaG_checkopenop (Instruction i)
func gCheckOpenOp(i Instruction) bool {
	// todo：gCheckOpenOp
	log.Println("gCheckOpenOp not implemented")
	return true
}
