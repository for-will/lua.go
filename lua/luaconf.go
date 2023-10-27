package golua

import (
	"fmt"
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

// NumberToStr
// 对应C函数：`lua_number2str(s,n)'
func NumberToStr(n LuaNumber) string {
	return fmt.Sprintf(LUA_NUMBER_FMT, n)
}

// LUAI_MAXCCALLS is the maximum depth for nested C calls (short) and
// syntactical nested non-terminals in a program.
const LUAI_MAXCCALLS = 200

// LUA_COMPAT_VARARG controls compatibility whith old vararg feature.
// CHNAGE it to undefined as soon as your programs use only '...' to
// access vararg parameters (instead of the old 'arg' table).
const LUA_COMPAT_VARARG = true
