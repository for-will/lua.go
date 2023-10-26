package golua

import (
	"fmt"
)

const (
	LUAL_BUFFERSIZE = 1024
)

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
