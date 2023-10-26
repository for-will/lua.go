package golua

import "math"

type lu_byte = uint8

// #define MAX_SIZET	((size_t)(~(size_t)0)-2)

const MAX_SIZET = (^uint32(0)) - 2

const MAX_INT = math.MaxInt32 - 2

const LUA_MINBUFFER = 32 /* minimum size fo string buffer */

func LuaAssert(c bool) {
	// todo
}
func CheckExp(c bool) {
	// todo
	if !c {
		panic("CheckExp Failed")
	}
}

// Instruction 对应C类型`Instruction`
// type for virtual-machine instructions
// must be an unsigned with (at least) 4 bytes (see details in lopcodes.h)
type Instruction = int32

// CondHardStackTests
// 对应C函数：condhardstacktests(x)
func CondHardStackTests() bool {
	// return HARDSTACKTESTS
	return false
}
