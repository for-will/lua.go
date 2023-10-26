package golua

import "unsafe"

// NewLClosure
// 对应C函数：`Closure *luaF_newLclosure (lua_State *L, int nelems, Table *e)'
func NewLClosure(L *LuaState, nelems int, e *Table) *LClosure {
	c := &LClosure{
		upVals: make([]*UpVal, nelems),
	}
	// todo: luaC_link(L, obj2gco(c), LUA_TFUNCTION);
	c.isC = false
	c.env = e
	c.nUpValues = lu_byte(nelems)
	return c
}

// NewUpVal
// 对应C函数：`UpVal *luaF_newupval (lua_State *L)'
func NewUpVal(L *LuaState) *UpVal {
	uv := &UpVal{}
	// todo: luaC_link(L, obj2gco(uv), LUA_TUPVAL)
	uv.v = &uv.value
	uv.v.SetNil()
	return uv
}

// 对应C函数：`void luaF_close (lua_State *L, StkId level)'
func (L *LuaState) fClose(level StkId) {
	// todo: fClose (这个函数还没有实现)
	// g := L.G()
	for L.openUpval != nil {
		uv := L.openUpval.ToUpval()
		if uintptr(unsafe.Pointer(uv.v)) < uintptr(unsafe.Pointer(level)) {
			break
		}
		LuaAssert(!uv.IsBlack() && uv.v != &uv.value)
		L.openUpval = uv.next /* remove from `open' list */
	}
}
