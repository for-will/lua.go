package golua

import (
	"math"
)

const (
	LUAL_BUFFERSIZE = 1024
)

// ApiCheck
// 对应C函数：luai_apicheck(L,o)
func ApiCheck(L *LuaState, o bool) {
	// 不做检查
}

const LUA_NUMBER_FMT = "%.14g"

const LUA_QS = "'%s'"

// // NumberToStr
// // 对应C函数：`lua_number2str(s,n)'
// func NumberToStr(n LuaNumber) string {
// 	return fmt.Sprintf(LUA_NUMBER_FMT, n)
// }

/* The luai_num* macros define the primitive operations over numbers. */
func luai_numadd(a, b LuaNumber) LuaNumber {
	return a + b
}
func luai_numsub(a, b LuaNumber) LuaNumber {
	return a - b
}
func luai_nummul(a, b LuaNumber) LuaNumber {
	return a * b
}
func luai_numdiv(a, b LuaNumber) LuaNumber {
	return a / b
}
func luai_nummod(a, b LuaNumber) LuaNumber {
	return a - math.Floor(a/b)*b
}
func luai_numpow(a, b LuaNumber) LuaNumber {
	return math.Pow(a, b)
}
func luai_numunm(a LuaNumber) LuaNumber {
	return -a
}
func luai_numeq(a, b LuaNumber) bool {
	return a == b
}
func luai_numlt(a, b LuaNumber) bool {
	return a < b
}
func luai_numle(a, b LuaNumber) bool {
	return a <= b
}
func luai_numisnan(a LuaNumber) bool {
	return !luai_numeq(a, a)
}

// LUAI_MAXCCALLS is the maximum depth for nested C calls (short) and
// syntactical nested non-terminals in a program.
const LUAI_MAXCCALLS = 200

// LUAI_MAXVARS is the maximum number of local variables per function
// (must be smaller than 250).
const LUAI_MAXVARS = 200

// LUAI_MAXUPVALUES is the maximum number of upvalues per function
// (must be smaller than 250).
const LUAI_MAXUPVALUES = 60

// LUA_COMPAT_VARARG controls compatibility whith old vararg feature.
// CHNAGE it to undefined as soon as your programs use only '...' to
// access vararg parameters (instead of the old 'arg' table).
const LUA_COMPAT_VARARG = true

// LUA_COMPAT_LSTR controls compatibility with old long string nesting
// facility.
// CHANGE it to 2 if you want the old behaviour, or undefine it to turn
// off the advisory error when nesting [[...]].
const LUA_COMPAT_LSTR = 1

type (
	LUAI_UINT32 = uint32
	LUAI_INT32  = int32
	LUAI_UMEM   = int
	LUAI_MEM    = uintptr
)

// LUAI_GCPAUSE defines the default pause between garbage-collector cycles
// as a percentage.
// CHANGE it if you want the GC to run faster or slower (higher values
// mean larger pauses which mean slower collection.) You can also change
// this value dynamically.
const LUAI_GCPAUSE = 200 /* 200% (wait memory to double before next GC) */

// LUAI_GCMUL defines the default speed of garbage collection relative to
// memory allocation as a percentage.
// CHANGE it if you want to change the granularity of th garbage
// collection. (Higher values mean coarser collections. 0 represents
// infinity, where each step performs a full collection.) You can also
// change this value dynamically.
const LUAI_GCMUL = 200 /* GC runs 'twice the speed' of memory allocation */

// luai_userstate* allow user-specific actions on threads.
// CHANGE them if you defined LUAI_EXTREASPACE and need to do something
// extra when a thread is created/deleted/resumed/yielded.
var (
	LUAIUserStateOpen   = func(L *LuaState) {}
	LUAIUserStateClose  = func(L *LuaState) {}
	LUAIUserStateThread = func(L *LuaState) {}
	LUAIUserStateFree   = func(L *LuaState) {}
	LUAIUserStateResume = func(L *LuaState) {}
	LUAIUserStateYield  = func(L *LuaState) {}
)

const SHRT_MAX = math.MaxInt16

const DEBUG = true /* 设置为true将会打印字节码的反汇编信息等其它调试信息 */
