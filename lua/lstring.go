package golua

// 对应C函数：`luaS_newliteral(L, s)'
func (L *LuaState) sNewLiteral(s string) *TString {
	return L.sNewLStr([]byte(s))
}
