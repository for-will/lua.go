package golua

const MEMERRMSG = "not enough memory"

// 对应C函数：`void *luaM_toobig (lua_State *L)'
func (L *LuaState) mTooBig() interface{} {
	L.gRunError("memory allocation error: block too big")
	return nil /* to avoid warnings */
}
