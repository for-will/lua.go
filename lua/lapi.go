package golua

import (
	"unsafe"
)

// PCall
// 对应C函数：`LUA_API int lua_pcall (lua_State *L, int nargs, int nresults, int errfunc)'
func (L *LuaState) PCall(nargs int, nresults int, errFunc int) int {
	var c callS
	L.Lock()
	L.apiCheckNElems(nargs + 1)
	L.checkResults(nargs, nresults)

	var funcIdx int
	if errFunc == 0 {
		funcIdx = 0
	} else {
		o := index2adr(L, errFunc)
		L.apiCheckValidIndex(o)
		funcIdx = savestack(L, o)
	}
	c.fun = L.AtTop(-(nargs + 1)) /* function to be called */
	c.nResults = nresults
	status := L.dPCall(f_call, &c, savestack(L, c.fun), funcIdx)
	// todo: adjustresults(L, nresults);
	L.Unlock()
	return status
}

// 对应C函数：`static void f_call (lua_State *L, void *ud)'
func f_call(L *LuaState, ud interface{}) {
	// todo: f_call
}

// Execute a protected call
// 对应C结构体：`struct CallS'
type callS struct { /* data to `f_call' */
	fun      StkId
	nResults int
}

// PushLString
// 对应C函数：`LUA_API void lua_pushlstring (lua_State *L, const char *s, size_t len)'
func (L *LuaState) PushLString(s []byte) {
	L.Lock()
	// todo: luaC_checkGC(L);
	L.Top().SetString(L, L.sNewLStr(s))
	L.apiIncrTop()
	L.Unlock()
}

// PushLiteral
// 对应C函数：`lua_pushliteral(L, s)'
func (L *LuaState) PushLiteral(s string) {
	L.PushLString([]byte(s))
}

func (L *LuaState) PushFString(format string, args ...interface{}) []byte {
	L.Lock()
	// todo: luaC_checkGC(L);
	ret := oPushVfString(L, []byte(format), args)
	L.Unlock()
	return ret
}

// 对应C函数：`api_checknelems(L, n)'
func (L *LuaState) apiCheckNElems(n int) {
	ApiCheck(L, n <= L.top-L.base)
}

// 对应C函数：`api_checkvalidindex(L, i)'
func (L *LuaState) apiCheckValidIndex(i StkId) {
	ApiCheck(L, i != LuaObjNil)
}

// 对应C函数：`api_incr_top(L)'
func (L *LuaState) apiIncrTop() {
	ApiCheck(L, L.top < L.CI().top)
	L.top++
}

// 对应C函数：`checkresults(L,na,nr)'
func (L *LuaState) checkResults(na int, nr int) {
	ApiCheck(L, nr == LUA_MULTRET || L.CI().top-L.top >= nr-na)
}

// ToLString
// 对应C函数：`LUA_API const char *lua_tolstring (lua_State *L, int idx, size_t *len)'
func (L *LuaState) ToLString(idx int) (b []byte, len int) {
	o := index2adr(L, idx)
	if o.IsNil() {
		L.Lock()            /*`luaV_tostring' may create a new string */
		if !o.ToString(L) { /* conversion failed? */
			L.Unlock()
			return nil, 0
		}
		// todo: luaC_checkGC(L);
		o = index2adr(L, idx) /* previous call may reallocate the stack */
		L.Unlock()
	}
	s := o.StringValue()
	return s.Bytes, s.Len
}

// Load
// 对应C函数：`LUA_API int lua_load (lua_State *L, lua_Reader reader, void *data, const char *chunkname)'
func (L *LuaState) Load(reader LuaReadFunc, data interface{}, chunkName []byte) int {
	var z ZIO
	L.Lock()
	if chunkName == nil {
		chunkName = []byte("?")
	}
	z.Init(L, reader, data)
	status := L.dProtectedParser(&z, chunkName)
	L.Unlock()
	return status
}

func index2adr(L *LuaState, idx int) *TValue {
	p, _ := index2addr(L, idx)
	return p
}

func index2addr(L *LuaState, idx int) (*TValue, int) {
	if idx > 0 {
		ApiCheck(L, idx <= L.CI().top-L.base)
		if L.base+idx-1 >= L.top {
			return LuaObjNil, -1
		} else {
			return L.AtBase(idx - 1), idx - 1
		}
	} else if idx > LUA_REGISTRYINDEX {
		ApiCheck(L, idx != 0 && -idx <= L.top-L.base)
		return L.AtTop(idx), L.top + idx
	} else { /* pseudo-indices */
		switch idx {
		case LUA_REGISTRYINDEX:
			return L.Registry(), -1
		case LUA_ENVIRONINDEX:
			cl := L.CurrFunc().C()
			L.env.SetTable(L, cl.env)
			return &L.env, -1
		case LUA_GLOBALSINDEX:
			return L.GlobalTable(), -1
		default:
			cl := L.CurrFunc().C()
			idx = LUA_GLOBALSINDEX - idx
			if idx <= int(cl.nUpValues) {
				return &cl.upValue[idx-1], -1
			} else {
				return LuaObjNil, -1
			}
		}
	}
}

func adr2idx(L *LuaState, p *TValue) int {
	off := uintptr(unsafe.Pointer(p)) - uintptr(unsafe.Pointer(&L.stack[0]))
	return int(off / unsafe.Sizeof(TValue{}))
}

func (L *LuaState) Remove(idx int) {
	L.Lock()
	p, i := index2addr(L, idx)
	i++
	for i < L.top {
		p2 := L.AtBase(i)
		SetObj(L, p, p2)
		p = p2
		i++
	}
	L.top--
	L.Unlock()
}
