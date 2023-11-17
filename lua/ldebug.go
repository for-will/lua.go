package golua

import (
	"log"
)

// ResetHookCount
// 对应C函数：`resethookcount(L)'
func ResetHookCount(L *LuaState) {
	L.hookCount = L.baseHootCount
}

// DbgRunError
// 对应C函数：`void luaG_runerror (lua_State *L, const char *fmt, ...)'
func (L *LuaState) DbgRunError(format string, args ...interface{}) {
	// todo: gRunError
	log.Printf(format, args...)
}

func (L *LuaState) gConcatError(p1 StkId, p2 StkId) {
	// todo: gConcatError
	log.Println("concat error")
}

// 对应C函数：`void luaG_aritherror (lua_State *L, const TValue *p1, const TValue *p2)'
func (L *LuaState) gArithError(p1 StkId, p2 StkId) {
	// todo: gArithError
	log.Println("arith error")
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

// 对应C函数：`int luaG_checkopenop (Instruction i)'
func gCheckOpenOp(i Instruction) bool {
	// todo：gCheckOpenOp
	log.Println("gCheckOpenOp not implemented")
	return true
}

// 对应C函数：`int luaG_ordererror (lua_State *L, const TValue *p1, const TValue *p2) '
func (L *LuaState) gOrderError(p1 *TValue, p2 *TValue) bool {
	// todo：gOrderError
	log.Println("gOrderError not implemented")
	return false
}

// 对应C函数：`int luaG_checkcode (const Proto *pt)'
func (p *Proto) gCheckCode() bool {
	// todo: gCheckCode
	log.Println("gCheckCode not implemented")
	return true
}
