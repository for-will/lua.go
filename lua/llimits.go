package golua

import (
	"log"
	"math"
)

type lu_mem = LUAI_UMEM

type lu_byte = uint8

// #define MAX_SIZET	((size_t)(~(size_t)0)-2)

const (
	MAX_SIZET = (^uint32(0)) - 2
	MAX_INT   = math.MaxInt32 - 2
)

const (
	MAXSTACK      = 250 /* maximum stack for Lua function */
	MINSTRTABSIZE = 32  /* minimum size for the string table (must be power of 2) */
	LUA_MINBUFFER = 32  /* minimum size fo string buffer */
)

// 对应C函数：luai_threadyield(L)
func (L *LuaState) iThreadYield() {
	L.Unlock()
	L.Lock()
}

func LuaAssert(c bool) {
	// todo: LuaAssert
	if !c {
		log.Panic("assert failed")
	}
}
func CheckExp(c bool) {
	// todo: CheckExp
	if !c {
		panic("CheckExp Failed")
	}
}

// CondHardStackTests
// 对应C函数：condhardstacktests(x)
func CondHardStackTests() bool {
	// return HARDSTACKTESTS
	return false
}
