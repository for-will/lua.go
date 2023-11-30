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
	L.IncrTop()
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

// IncrTop 对应C函数：`api_incr_top(L)'
func (L *LuaState) IncrTop() {
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
func (L *LuaState) checkResults(args int, results int) {
	ApiCheck(L, results == LUA_MULTRET || L.CI().top-L.top >= results-args)
}

// Call
// 对应C函数：`LUA_API void lua_call (lua_State *L, int nargs, int nresults)'
func (L *LuaState) Call(nArgs int, nResults int) {
	L.Lock()
	L.apiCheckNElems(nArgs + 1)
	L.checkResults(nArgs, nResults)
	var f = L.AtTop(-(nArgs + 1))
	L.dCall(f, nResults)
	L.adjustResults(nResults)
	L.Unlock()
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
			return L.AtBase(idx - 1), L.base + idx - 1
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

// Remove
// 对应C函数：`LUA_API void lua_remove (lua_State *L, int idx)'
func (L *LuaState) Remove(idx int) {
	L.Lock()
	p, i := index2addr(L, idx)
	i++
	for i < L.top {
		var p2 = L.At(i)
		SetObj(L, p, p2)
		p = p2
		i++
	}
	L.top--
	L.Unlock()
}

// Insert
// 对应C函数：`LUA_API void lua_insert (lua_State *L, int idx)'
func (L *LuaState) Insert(idx int) {
	L.Lock()
	var p, pi = index2addr(L, idx)
	L.apiCheckValidIndex(p)
	for q := L.top; q > pi; q-- {
		L.At(q).SetObj(L, L.At(q-1))
	}
	p.SetObj(L, L.Top())
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
	L.IncrTop()
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

// RawSet
// 对应C函数：`LUA_API void lua_rawset (lua_State *L, int idx) '
func (L *LuaState) RawSet(idx int) {
	L.Lock()
	L.apiCheckNElems(2)
	var t = index2adr(L, idx)
	L.apiCheck(t.IsTable())
	t.TableValue().Set(L, L.AtTop(-2)).SetObj(L, L.AtTop(-1))
	L.cBarrierT(t.TableValue(), L.AtTop(-1))
	L.top -= 2
	L.Unlock()
}

// SetMetaTable
// 对应C函数：`LUA_API int lua_setmetatable (lua_State *L, int objindex)'
func (L *LuaState) SetMetaTable(objIndex int) int {
	var mt *Table
	L.Lock()
	L.apiCheckNElems(1)
	var obj = index2adr(L, objIndex)
	L.apiCheckValidIndex(obj)
	t1 := L.AtTop(-1)
	if t1.IsNil() {
		mt = nil
	} else {
		L.apiCheck(t1.IsTable())
		mt = t1.TableValue()
	}
	switch obj.gcType() {
	case LUA_TTABLE:
		obj.TableValue().metatable = mt
		if mt != nil {
			L.cObjBarrierT(obj.TableValue(), mt)
		}
	case LUA_TUSERDATA:
		obj.UdataValue().metatable = mt
		if mt != nil {
			L.cObjBarrier(obj.UdataValue(), mt)
		}
	default:
		L.G().mt[obj.gcType()] = mt
	}
	L.top--
	L.Unlock()
	return 1
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

// PushValue
// 对应C函数：`LUA_API void lua_pushvalue (lua_State *L, int idx)'
func (L *LuaState) PushValue(idx int) {
	L.Lock()
	L.Top().SetObj(L, index2adr(L, idx))
	L.IncrTop()
	L.Unlock()
}

// RawGet
// 对应C函数：`LUA_API void lua_rawget (lua_State *L, int idx)'
func (L *LuaState) RawGet(idx int) {
	L.Lock()
	var t = index2adr(L, idx)
	ApiCheck(L, t.IsTable())
	var t1 = L.AtTop(-1)
	t1.SetObj(L, t.TableValue().Get(t1))
	L.Unlock()
}

// RawGetI
// 对应C函数：`LUA_API void lua_rawgeti (lua_State *L, int idx, int n) '
func (L *LuaState) RawGetI(idx int, n int) {
	L.Lock()
	var o = index2adr(L, idx)
	L.apiCheck(o.IsTable())
	L.Top().SetObj(L, o.TableValue().GetNum(n))
	L.IncrTop()
	L.Unlock()
}

// CreateTable
// 对应C函数：`LUA_API void lua_createtable (lua_State *L, int narray, int nrec)'
func (L *LuaState) CreateTable(nArray, nRec int) {
	L.Lock()
	L.cCheckGC()
	L.Top().SetTable(L, L.hNew(nArray, nRec))
	L.IncrTop()
	L.Unlock()
}

// GetMetaTable
// 对应C函数：`LUA_API int lua_getmetatable (lua_State *L, int objindex)'
func (L *LuaState) GetMetaTable(objIndex int) int {
	L.Lock()
	var obj = index2adr(L, objIndex)
	var mt *Table
	switch obj.gcType() {
	case LUA_TTABLE:
		mt = obj.TableValue().metatable
	case LUA_TUSERDATA:
		mt = obj.UdataValue().metatable
	default:
		mt = L.G().mt[obj.tt]
	}
	var res int
	if mt == nil {
		res = 0
	} else {
		L.Top().SetTable(L, mt)
		L.IncrTop()
		res = 1
	}
	L.Unlock()
	return res
}

// NewTable
// 对应C函数：`lua_newtable(L)'
func (L *LuaState) NewTable() {
	L.CreateTable(0, 0)
}

// SetTable stack[idx][stack[-2]] = stack[-1]
// 对应C函数：`LUA_API void lua_settable (lua_State *L, int idx)'
func (L *LuaState) SetTable(idx int) {
	L.Lock()
	L.apiCheckNElems(2)
	var t = index2adr(L, idx)
	L.apiCheckValidIndex(t)
	L.vSetTable(t, L.AtTop(-2), L.AtTop(-2))
	L.top -= 2 /* pop index and value */
	L.Unlock()
}

// GetField
// 对应C函数：`LUA_API void lua_getfield (lua_State *L, int idx, const char *k)'
func (L *LuaState) GetField(idx int, k string) {
	L.Lock()
	var t = index2adr(L, idx)
	L.apiCheckValidIndex(t)
	var key TValue
	key.SetString(L, L.sNewLiteral(k))
	L.vGetTable(t, &key, L.Top())
	L.IncrTop()
	L.Unlock()
}

// PushVFString
// 对应C函数：`LUA_API const char *lua_pushvfstring (lua_State *L, const char *fmt, va_list argp)'
func (L *LuaState) PushVFString(format string, args []interface{}) []byte {
	L.Lock()
	L.cCheckGC()
	var ret = L.oPushVfString([]byte(format), args)
	L.Unlock()
	return ret
}

// LuaNext
// 对应C函数：`LUA_API int lua_next (lua_State *L, int idx)'
func (L *LuaState) LuaNext(idx int) bool {
	L.Lock()
	var t = index2adr(L, idx)
	L.apiCheck(t.IsTable())
	var more = t.TableValue().hNext(L, L.AtTop(-1))
	if more {
		L.IncrTop()
	} else { /* no more elements */
		L.top -= 1 /* remove key */
	}
	L.Unlock()
	return more
}

// Concat
// 对应C函数：`LUA_API void lua_concat (lua_State *L, int n)'
func (L *LuaState) Concat(n int) {
	L.Lock()
	L.apiCheckNElems(n)
	if n >= 2 {
		L.cCheckGC()
		L.vConcat(n, L.top-L.base-1)
		L.top -= n - 1
	} else if n == 0 { /* push empty string */
		L.Top().SetString(L, L.sNewLiteral(""))
		L.IncrTop()
	}
	/* else n == 1; nothing to do */
	L.Unlock()
}

// 对应C函数：`LUA_API int lua_error (lua_State *L)'
func (L *LuaState) Error() int {
	L.Lock()
	L.apiCheckNElems(1)
	L.gErrorMsg()
	L.Unlock()
	return 0
}

/* push functions (C -> stack) */

// PushNil
// 对应C函数：`LUA_API void lua_pushnil (lua_State *L)'
func (L *LuaState) PushNil() {
	L.Lock()
	L.Top().SetNil()
	L.IncrTop()
	L.Unlock()
}

// PushInteger
// 对应C函数：`LUA_API void lua_pushinteger (lua_State *L, lua_Integer n)'
func (L *LuaState) PushInteger(n LuaInteger) {
	L.Lock()
	L.Top().SetNumber(LuaNumber(n))
	L.IncrTop()
	L.Unlock()
}

// PushBoolean
// 对应C函数：`LUA_API void lua_pushboolean (lua_State *L, int b)'
func (L *LuaState) PushBoolean(b bool) {
	L.Lock()
	L.Top().SetBoolean(b)
	L.IncrTop()
	L.Unlock()
}

// ToInteger
// 对应C函数：`LUA_API lua_Integer lua_tointeger (lua_State *L, int idx)'
func (L *LuaState) ToInteger(idx int) LuaInteger {
	var o = index2adr(L, idx)
	var n TValue
	if tonumber(o, &n) {
		var num = o.NumberValue()
		return lua_number2integer(num)
	} else {
		return 0
	}
}

// IsNumber
// 对应C函数：`LUA_API int lua_isnumber (lua_State *L, int idx)'
func (L *LuaState) IsNumber(idx int) bool {
	var n TValue
	var o = index2adr(L, idx)
	return tonumber(o, &n)
}

// NewUserData
// 对应C函数：`LUA_API void *lua_newuserdata (lua_State *L, size_t size) '
func (L *LuaState) NewUserData(size int) interface{} {
	L.Lock()
	L.cCheckGC()
	var u = L.sNewUData(size, L.getCurrEnv())
	L.Top().SetUserData(L, u)
	L.IncrTop()
	L.Unlock()
	return u.data
}
