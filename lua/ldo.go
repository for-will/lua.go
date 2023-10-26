package golua

import "unsafe"

// PFunc
// type of protected functions, to be ran by `runprotected'
// 对应C类型：`typedef void (*Pfunc) (lua_State *L, void *ud)'
type PFunc func(L *LuaState, ud interface{})

// 对应C函数：`int luaD_pcall (lua_State *L, Pfunc func, void *u,
//                ptrdiff_t old_top, ptrdiff_t ef)'
func (L *LuaState) dPCall(fun PFunc, u interface{}, oldTopIdx int, ef int) int {
	var (
		oldNCCalls    = L.nCCalls
		oldCi         = L.ci
		oldAllowHooks = L.allowHook
		oldErrFunc    = L.errFunc
	)
	L.errFunc = ef
	status := L.dRawRunProtected(fun, u)
	if status != 0 { /* an error occurred? */
		oldTop := &L.stack[oldTopIdx]
		L.fClose(oldTop) /* close eventual pending closures */
		L.dSetErrorObj(status, oldTopIdx)
		L.nCCalls = oldNCCalls
		L.ci = oldCi
		L.base = L.CI().base
		L.savedPc = L.CI().savedPc
		L.allowHook = oldAllowHooks
	}
	L.errFunc = oldErrFunc
	return status
}

// 对应C函数：`int luaD_rawrunprotected (lua_State *L, Pfunc f, void *ud)'
func (L *LuaState) dRawRunProtected(f PFunc, ud interface{}) (status int) {
	var lj LuaLongJmp
	lj.status = 0
	lj.previous = L.errorJmp /* chain new error handler */
	L.errorJmp = &lj
	defer func() {
		if err := recover(); err != nil {
			if L.errorJmp == &lj { /* 这里有必要吗？*/
				L.errorJmp = lj.previous
				status = lj.status
			} else {
				panic(err)
			}
		}
	}()
	f(L, ud)
	L.errorJmp = lj.previous /* restore old error handler */
	return lj.status
}

// 对应C函数：`void luaD_seterrorobj (lua_State *L, int errcode, StkId oldtop)'
func (L *LuaState) dSetErrorObj(errCode int, oldTopIdx int) {
	oldTop := &L.stack[oldTopIdx]
	switch errCode {
	case LUA_ERRMEM:
		oldTop.SetString(L, L.sNewLiteral(MEMERRMSG))
	case LUA_ERRERR:
		oldTop.SetString(L, L.sNewLiteral("error in error handling"))
	case LUA_ERRSYNTAX, LUA_ERRRUN:
		SetObj(L, oldTop, L.AtTop(-1)) /* error message on current top */
	}
	L.top = oldTopIdx + 1
}

// Execute a protected parser

// SParser data to `f_parser'
type SParser struct {
	z    *ZIO
	buff MBuffer /* buffer to be used by the scanner */
	name []byte
}

// 同C函数 `static void f_parser (lua_State *L, void *ud)'
func parser(L *LuaState, ud interface{}) {

	var tf *Proto
	p := ud.(*SParser)
	c := p.z.Lookahead()
	// todo: luaC_checkGC(L);
	if c == int([]byte(LUA_SIGNATURE)[0]) {
		tf = L.Undump(p.z, &p.buff, p.name)
	} else {
		tf = L.YParser(p.z, &p.buff, p.name)
	}
	cl := NewLClosure(L, tf.nups, L.GlobalTable().TableValue())
	cl.p = tf
	for i := 0; i < tf.nups; i++ {
		cl.upVals[i] = NewUpVal(L)
	}
	L.Top().SetClosure(L, cl)
	L.IncTop()
}

// 对应C函数：`int luaD_protectedparser (lua_State *L, ZIO *z, const char *name)'
func (L *LuaState) dProtectedParser(z *ZIO, name []byte) int {
	var p SParser
	p.z = z
	p.name = name
	p.buff.Init() /* 在go语言中基实不必做这一步的初始化 */
	status := L.dPCall(parser, &p, L.top, L.errFunc)
	p.buff.Free()
	return status
}

// 与C中实现不同的地方：存放在stack中的下标，而不是相对的地址偏移。
// 同C函数：savestack(L,p)
func savestack(L *LuaState, p StkId) int {
	return adr2idx(L, p)
}

// 对应C函数：`saveci(L,p)'
func saveci(L *LuaState, p *CallInfo) uintptr {
	return uintptr(unsafe.Pointer(p)) - uintptr(unsafe.Pointer(&L.baseCi[0]))
}

// LuaLongJmp
// chain list of long jump buffers
// GO没有long jump，先写在这里
// 对应C结构体：`struct lua_longjmp'
type LuaLongJmp struct {
	previous *LuaLongJmp
	b        [36]int
	status   int
}
