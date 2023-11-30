package lib

import golua "luar/lua"

// OpenLibs
// 对应C函数：`LUALIB_API void luaL_openlibs (lua_State *L)'
func OpenLibs(L *LuaState) {
	var libs = []golua.LReg{
		{"", LuaOpenBase},
	}
	for _, l := range libs {
		L.PushCFunction(l.Func)
		L.PushString(l.Name)
		L.Call(1, 0)
	}
}
