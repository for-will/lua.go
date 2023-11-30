package golua

import (
	"luar/lua/mem"
	"unsafe"
)

// 对应C函数：`luaS_newliteral(L, s)'
func (L *LuaState) sNewLiteral(s string) *TString {
	return L.sNewStr([]byte(s))
}

// 对应C函数：`Udata *luaS_newudata (lua_State *L, size_t s, Table *e) '
func (L *LuaState) sNewUData(sz int, e *Table) *Udata {
	if sz > int(MAX_SIZET)-int(unsafe.Sizeof(Udata{})) {
		mem.ErrTooBig(L)
	}
	var u = &Udata{}
	u.marked = L.G().cWhite() /* is not finalized */
	u.tt = LUA_TUSERDATA
	u.len = sz
	u.metatable = nil
	u.env = e
	u.data = make([]byte, sz)
	/* chain it on udata list (after main thread) */
	u.next = L.G().mainThread.next
	L.G().mainThread.next = u
	return u
}
