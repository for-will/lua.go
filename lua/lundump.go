package golua

// Undump load precompiled chuck
// 对应C函数：`Proto* luaU_undump (lua_State* L, ZIO* Z, Mbuffer* buff, const char* name)'
func (L *LuaState) Undump(Z *ZIO, buff *MBuffer, name []byte) *Proto {
	// todo
	return nil
}
