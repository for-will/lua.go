package golua

import "unsafe"

// fNewLClosure
// 对应C函数：`Closure *luaF_newLclosure (lua_State *L, int nelems, Table *e)'
func (L *LuaState) fNewLClosure(nelems int, e *Table) *LClosure {
	c := &LClosure{
		upVals: make([]*UpVal, nelems),
	}
	// todo: luaC_link(L, obj2gco(c), LUA_TFUNCTION);
	c.isC = false
	c.env = e
	c.nUpValues = lu_byte(nelems)
	return c
}

// fFindUpVal
// 对应C函数：`UpVal *luaF_findupval (lua_State *L, StkId level)'
func (L *LuaState) fFindUpVal(level StkId) *UpVal {
	var g = L.G()
	var pp = &L.openUpval
	for *pp != nil {
		var p = (*pp).ToUpval()
		if !(uintptr(unsafe.Pointer(p.v)) >= uintptr(unsafe.Pointer(level))) {
			break
		}
		if p.v == level { /* found a corresponding up-value? */
			if isdead(g, p) { /* is it dead? */
				p.ChangeWhite() /* ressurect it */
				return p
			}
		}
		pp = &p.next
	}
	var uv = &UpVal{} /* not found: create a new one */
	uv.tt = LUA_TUPVAL
	uv.marked = g.cWhite()
	uv.v = level  /* current value lives in the stack */
	uv.next = *pp /* chain it in the proper position */
	*pp = uv
	uv.l.prev = &g.uvHead
	uv.l.next = g.uvHead.l.next
	g.uvHead.l.next.l.prev = uv
	g.uvHead.l.next = uv
	LuaAssert(uv.l.next.l.prev == uv && uv.l.prev.l.next == uv)
	return uv
}

// fNewUpVal
// 对应C函数：`UpVal *luaF_newupval (lua_State *L)'
func fNewUpVal(L *LuaState) *UpVal {
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

// 对应C函数：`Proto *luaF_newproto (lua_State *L)'
func (L *LuaState) fNewProto() *Proto {
	f := &Proto{
		k:               nil,
		code:            nil,
		p:               nil,
		lineInfo:        nil,
		locVars:         nil,
		upValues:        nil,
		source:          nil,
		sizeUpValues:    0,
		sizeK:           0,
		sizeCode:        0,
		sizeLineInfo:    0,
		sizeP:           0,
		sizeLocVars:     0,
		lineDefined:     0,
		lastLineDefined: 0,
		gcList:          nil,
		nUps:            0,
		numParams:       0,
		isVarArg:        0,
		maxStackSize:    0,
	}
	// todo: luaC_link(L, obj2gco(f), LUA_TPROTO);
	return f
}
