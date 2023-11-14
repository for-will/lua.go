package golua

import (
	"reflect"
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
	L.adjustResults(nresults)
	L.Unlock()
	return status
}

// 对应C函数：`static void f_call (lua_State *L, void *ud)'
func f_call(L *LuaState, ud interface{}) {
	c := ud.(*callS)
	L.dCall(c.fun, c.nResults)
}

// Execute a protected call
// 对应C结构体：`struct CallS'
type callS struct { /* data to `f_call' */
	fun      StkId
	nResults int
}

// PushString
// 对应C函数：`LUA_API void lua_pushlstring (lua_State *L, const char *s, size_t len)'
func (L *LuaState) PushString(s string) {
	L.Lock()
	L.cCheckGC()
	L.Top().SetString(L, L.sNewStr([]byte(s)))
	L.apiIncrTop()
	L.Unlock()
}

// PushLiteral
// 对应C函数：`lua_pushliteral(L, s)'
func (L *LuaState) PushLiteral(s string) {
	L.PushString(s)
}

func (L *LuaState) PushFString(format string, args ...interface{}) []byte {
	L.Lock()
	L.cCheckGC()
	ret := L.oPushVfString([]byte(format), args)
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

// 对应C函数：`adjustresults(L,nres)'
func (L *LuaState) adjustResults(nres int) {
	if nres == LUA_MULTRET && L.top >= L.CI().top {
		L.CI().top = L.top
	}
}

// 对应C函数：`checkresults(L,na,nr)'
func (L *LuaState) checkResults(na int, nr int) {
	ApiCheck(L, nr == LUA_MULTRET || L.CI().top-L.top >= nr-na)
}

// ToLString
// 对应C函数：`LUA_API const char *lua_tolstring (lua_State *L, int idx, size_t *len)'
func (L *LuaState) ToLString(idx int) (b []byte, len int) {
	o := index2adr(L, idx)
	if !o.IsString() {
		L.Lock()             /*`luaV_tostring' may create a new string */
		if !o.vToString(L) { /* conversion failed? */
			L.Unlock()
			return nil, 0
		}
		L.cCheckGC()
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

// AtPanic
// 对应C函数：`LUA_API lua_CFunction lua_atpanic (lua_State *L, lua_CFunction panicf)'
func (L *LuaState) AtPanic(fPanic LuaCFunction) LuaCFunction {
	L.Lock()
	var old = L.G().panic
	L.G().panic = fPanic
	L.Unlock()
	return old
}

// PushCClosure
// 对应C函数：`LUA_API void lua_pushcclosure (lua_State *L, lua_CFunction fn, int n)'
func (L *LuaState) PushCClosure(fn LuaCFunction, n int) {
	L.Lock()
	L.cCheckGC()
	L.apiCheckNElems(n)
	var cl = L.fNewCClosure(n, L.getCurrEnv())
	cl.f = fn
	L.top -= n
	for n > 0 {
		n--
		cl.upValue[n].SetObj(L, L.AtTop(n))
	}
	L.Top().SetClosure(L, cl)
	// todo: LuaAssert(cl.IsWhite())
	L.apiIncrTop()
	L.Unlock()
}

// 对应C函数：`static Table *getcurrenv (lua_State *L) '
func (L *LuaState) getCurrEnv() *Table {
	if L.ci == 0 { /* no enclosing function? */
		return L.GlobalTable().TableValue() /* use global table as environment */
	} else {
		return L.CurrFunc().C().env
	}
}

// SetField
// 对应C函数：`LUA_API void lua_setfield (lua_State *L, int idx, const char *k)'
func (L *LuaState) SetField(idx int, k string) {
	L.Lock()
	L.apiCheckNElems(1)
	var t = index2adr(L, idx)
	L.apiCheckValidIndex(t)
	var key TValue
	key.SetString(L, L.sNew([]byte(k)))
	L.vSetTable(t, &key, L.Top().Ptr(-1))
	L.top-- /* pop value */
	L.Unlock()
}

// Type
// 对应C函数：`LUA_API int lua_type (lua_State *L, int idx)'
func (L *LuaState) Type(idx int) ttype {
	var o = index2adr(L, idx)
	if o == LuaObjNil {
		return LUA_TNONE
	}
	return o.gcType()
}

// IsString
// 对应C函数：`LUA_API int lua_isstring (lua_State *L, int idx)'
func (L *LuaState) IsString(idx int) bool {
	var t = L.Type(idx)
	return t == LUA_TSTRING || t == LUA_TNUMBER
}

// IsFunction
// 对应C函数：`lua_isfunction(L,n)'
func (L *LuaState) IsFunction(idx int) bool {
	return L.Type(idx) == LUA_TFUNCTION
}

// IsTable
// 对应C函数：`lua_istable(L,n)'
func (L *LuaState) IsTable(idx int) bool {
	return L.Type(idx) == LUA_TTABLE
}

// IsLightUserData
// 对应C函数：`lua_islightuserdata(L,n)'
func (L *LuaState) IsLightUserData(idx int) bool {
	return L.Type(idx) == LUA_TLIGHTUSERDATA
}

// IsNil
// 对应C函数：`lua_isnil(L,n)'
func (L *LuaState) IsNil(idx int) bool {
	return L.Type(idx) == LUA_TNIL
}

// IsBoolean
// 对应C函数：`lua_isboolean(L,n)'
func (L *LuaState) IsBoolean(idx int) bool {
	return L.Type(idx) == LUA_TBOOLEAN
}

// IsThread
// 对应C函数：`lua_isthread(L,n)'
func (L *LuaState) IsThread(idx int) bool {
	return L.Type(idx) == LUA_TTHREAD
}

// IsNone
// 对应C函数：`lua_isnone(L,n)'
func (L *LuaState) IsNone(idx int) bool {
	return L.Type(idx) == LUA_TNONE
}

// IsNoneOrNil
// 对应C函数：`lua_isnoneornil(L,n)'
func (L *LuaState) IsNoneOrNil(idx int) bool {
	return L.Type(idx) <= 0
}

// ToBoolean
// 对应C函数：`LUA_API int lua_toboolean (lua_State *L, int idx) '
func (L *LuaState) ToBoolean(idx int) bool {
	var o = index2adr(L, idx)
	return !o.IsFalse()
}

// ToUserData
// 对应C函数：`LUA_API void *lua_touserdata (lua_State *L, int idx)'
func (L *LuaState) ToUserData(idx int) interface{} {
	var o = index2adr(L, idx)
	switch o.gcType() {
	case LUA_TUSERDATA:
		return o.UdataValue()
	case LUA_TLIGHTUSERDATA:
		return o.PointerValue()
	default:
		return nil
	}
}

// ToPointer
// 对应C函数：`LUA_API const void *lua_topointer (lua_State *L, int idx) '
func (L *LuaState) ToPointer(idx int) unsafe.Pointer {
	var o = index2adr(L, idx)
	switch o.gcType() {
	case LUA_TTABLE:
		return unsafe.Pointer(o.TableValue())
	case LUA_TFUNCTION:
		return reflect.ValueOf(o.ClosureValue()).Addr().UnsafePointer()
	case LUA_TTHREAD:
		return unsafe.Pointer(o.ThreadValue())
	case LUA_TUSERDATA, LUA_TLIGHTUSERDATA:
		return reflect.ValueOf(L.ToUserData(idx)).Addr().UnsafePointer()
	default:
		return nil
	}
}

// TypeName
// 对应C函数：`LUA_API const char *lua_typename (lua_State *L, int t)'
func (L *LuaState) TypeName(t ttype) string {
	if t == LUA_TNONE {
		return "no value"
	}
	return LuaTTypeNames[t]
}
