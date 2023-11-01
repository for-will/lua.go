package golua

import "unsafe"

/* results from luaD_precall */
const (
	PCRLUA   = 0 /* initiated a call to a Lua function */
	PCRC     = 1 /* did a call to a C function */
	PCRYIELD = 2 /* C function yielded */
)

// PFunc
// type of protected functions, to be ran by `runprotected'
// 对应C类型：`typedef void (*Pfunc) (lua_State *L, void *ud)'
type PFunc func(L *LuaState, ud interface{})

// 对应C函数：`int luaD_pcall (lua_State *L, Pfunc func, void *u,
//
//	ptrdiff_t old_top, ptrdiff_t ef)'
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
	L.cCheckGC()
	if c == int([]byte(LUA_SIGNATURE)[0]) {
		tf = L.uUndump(p.z, &p.buff, p.name)
	} else {
		tf = L.YParser(p.z, &p.buff, p.name)
	}
	cl := L.fNewLClosure(tf.nUps, L.GlobalTable().TableValue())
	cl.p = tf
	for i := 0; i < tf.nUps; i++ {
		cl.upVals[i] = fNewUpVal(L)
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

// 与C中实现不同的地方：返回p在stack中的下标，而不是相对的地址偏移。
// 同C函数：savestack(L,p)
func savestack(L *LuaState, p StkId) int {
	return adr2idx(L, p)
}

// 与C中实现不同：n是stack的下标，而不是相对的地址偏移量。
// 同C函数：restorestack(L,n)
func restorestack(L *LuaState, n int) StkId {
	return &L.stack[n]
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
	// b        [36]int
	status int
}

// 对应C函数：`static StkId callrethooks (lua_State *L, StkId firstResult)'
func callrethooks(L *LuaState, firstResults int) int {
	fr := firstResults
	L.dCallHook(LUA_HOOKRET, -1)
	if L.CI().fIsLua() { /* Lua function? */
		for L.hookMask&LUA_MASKRET != 0 && L.CI().tailCalls != 0 { /* tail calls */
			L.CI().tailCalls--
			L.dCallHook(LUA_HOOKTAILRET, -1)
		}
	}
	return fr
}

// 对应C函数：`int luaD_poscall (lua_State *L, StkId firstResult)'
func (L *LuaState) dPoscall(firstResult int) int {
	if L.hookMask&LUA_MASKRET != 0 {
		firstResult = callrethooks(L, firstResult)
	}
	ci := L.CI()
	L.ci--
	res := ci.fn /* res == final position of 1st result */
	wanted := ci.nResults
	L.base = L.CI().base       /* restore base */
	L.savedPc = L.CI().savedPc /* restore savedpc */
	/* move results to correct place */
	var i int
	for i = wanted; i != 0 && firstResult < L.top; i-- {
		res.SetObj(L, &L.stack[firstResult])
		res = res.Ptr(1) // res++
		firstResult++
	}
	for ; i > 0; i-- {
		res.SetNil()
		res = res.Ptr(1) // res++
	}
	L.top = adr2idx(L, res)
	return wanted - LUA_MULTRET /* 0 iff wanted == LUA_MULTRET */
}

// Call a function (C or Lua). The function to be called is at *func.
// The arguments are on the stack, right after the function.
// When returns, all the results are on the stack, starting at the original
// function position.
// 对应C函数：`void luaD_call (lua_State *L, StkId func, int nResults) '
func (L *LuaState) dCall(fn StkId, nResults int) {
	L.nCCalls++
	if L.nCCalls >= LUAI_MAXCCALLS {
		if L.nCCalls == LUAI_MAXCCALLS {
			L.gRunError("C stack overflow")
		} else if L.nCCalls >= LUAI_MAXCCALLS+LUAI_MAXCCALLS>>3 {
			L.dThrow(LUA_ERRERR) /* error while handing stack error */
		}
	}
	if L.dPrecall(fn, nResults) == PCRLUA { /* is a Lua function? */
		L.vExecute(1) /* call it */
	}
	L.nCCalls--
	L.cCheckGC()
}

// 对应C函数：`void luaD_throw (lua_State *L, int errcode)'
func (L *LuaState) dThrow(errCode int) {
	if L.errorJmp != nil {
		L.errorJmp.status = errCode
		panic(errCode)
		// LUAI_THROW(L, L->errorJmp);
	} else {
		L.status = lu_byte(errCode)
		// todo: dThrow (还没有完全实现)
	}
}

// 对应C函数：`int luaD_precall (lua_State *L, StkId func, int nresults)'
func (L *LuaState) dPrecall(fn StkId, nResults int) int {
	if !fn.IsFunction() { /* `fun' is not a function? */
		fn = tryFuncTM(L, fn) /* check th `function' tag method */
	}
	funcr := savestack(L, fn)
	cl := fn.ClosureValue().L()
	L.CI().savedPc = L.savedPc
	if !cl.isC { /* Lua function? prepare its call */
		var (
			base int
			p    = cl.p
		)

		L.dCheckStack(int(p.maxStackSize))
		fn = restorestack(L, funcr)
		if p.isVarArg == 0 { /* no varargs? */
			base = funcr + 1
			if L.top > base+p.numParams {
				L.top = base + p.numParams
			}
		} else { /* vararg function */
			nargs := L.top - funcr - 1 // -1 因为函数本身占一个
			base = adjust_varargs(L, p, nargs)
			fn = restorestack(L, funcr) /* previous call may change the stack */
		}
		ci := L.baseCi[inc_ci(L)] /* now `enter' new function */
		ci.fn = fn
		ci.base = base
		L.base = base
		ci.top = L.base + int(p.maxStackSize)
		LuaAssert(ci.top <= L.stackLast)
		L.savedPc = &p.code[0] /* starting point */
		ci.tailCalls = 0
		ci.nResults = nResults
		for st := L.top; st < ci.top; st++ {
			L.stack[st].SetNil()
		}
		L.top = ci.top
		if L.hookMask&LUA_MASKCALL != 0 {
			L.savedPc = L.savedPc.Ptr(1) /* hooks assume 'pc' is already incremented */
			L.dCallHook(LUA_HOOKCALL, -1)
			L.savedPc = L.savedPc.Ptr(-1) /* correct 'pc' */
		}
		return PCRLUA
	} else { /* if is a C function, call it */
		L.dCheckStack(LUA_MINSTACK) /* ensure minimum stack size */
		ci := L.baseCi[inc_ci(L)]   /* now `enter' new function */
		ci.fn = restorestack(L, funcr)
		ci.base = funcr + 1
		L.base = ci.base
		ci.top = L.top + LUA_MINSTACK
		LuaAssert(ci.top <= L.stackLast)
		ci.nResults = nResults
		if L.hookMask&LUA_MASKCALL != 0 {
			L.dCallHook(LUA_HOOKCALL, -1)
		}
		L.Unlock()
		n := L.CurrFunc().C().f(L) /* do the actual call */
		L.Lock()
		if n < 0 { /* yielding? */
			return PCRYIELD
		} else {
			L.dPoscall(L.top - n)
			return PCRC
		}
	}
}

// 对应C函数：`static StkId tryfuncTM (lua_State *L, StkId func)'
func tryFuncTM(L *LuaState, fn StkId) StkId {
	tm := L.tGetTMByObj(fn, TM_CALL)
	funcr := savestack(L, fn)
	if !tm.IsFunction() {
		L.gTypeError(fn, "call")
	}
	/* Open a hole inside the stack at `fn' */
	for p := L.top; p > funcr; p-- {
		SetObj(L, &L.stack[p], &L.stack[p-1])
	}
	L.IncTop()
	fn = restorestack(L, funcr) /* previous call may change stack */
	SetObj(L, fn, tm)           /* tag method is the new function to be called */
	return fn
}

// 对应C函数：`void luaD_callhook (lua_State *L, int event, int line)'
func (L *LuaState) dCallHook(event int, line int) {
	// todo: dCallHook
	panic("not implemented")
}

// 对应C函数：`static StkId adjust_varargs (lua_State *L, Proto *p, int actual)'
func adjust_varargs(L *LuaState, p *Proto, actual int) int {
	var (
		nFixArgs        = p.numParams
		htab     *Table = nil
	)
	for ; actual < nFixArgs; actual++ {
		L.Top().SetNil()
		L.top++
	}
	if LUA_COMPAT_VARARG {
		if p.isVarArg&VARARG_NEEDSARG != 0 { /* compat. with old-style vararg? */
			nvar := actual - nFixArgs /* number of extra arguments */
			LuaAssert(p.isVarArg&VARARG_HASARG != 0)
			L.cCheckGC()
			htab = L.hNew(nvar, 1)      /* create `arg' table */
			for i := 0; i < nvar; i++ { /* put extra argumetns into `arg' table */
				l := htab.SetByNum(L, i+1)
				r := L.AtTop(-nvar + i)
				SetObj(L, l, r)
			}
			/* store counter in field `n' */
			htab.SetByStr(L, L.sNewLiteral("n")).SetNumber(LuaNumber(nvar))
		}
	}
	/* move fixed parameters to final position */
	fixed := L.AtTop(-actual) /* first fixed argument */
	base := L.top             /* final position of first argument */
	for i := 0; i < nFixArgs; i++ {
		L.PushObj(fixed.Ptr(i))
		fixed.Ptr(i).SetNil()
	}
	/* add `arg' parameter */
	if htab != nil {
		L.PushTable(htab)
		LuaAssert(htab.IsWhite())
	}
	return base
}

// 对应C函数：` inc_ci(L)'
func inc_ci(L *LuaState) int {
	if L.ci == L.endCi {
		return growCI(L)
	} else {
		if CondHardStackTests() {
			L.dReallocCI(L.sizeCi)
		}
		L.ci++
		return L.ci
	}
}

// 这种static的C函数，只有一个地方被调用，可以放到被调用的地方，写成一个匿名函数。
// 对应C函数：`static CallInfo *growCI (lua_State *L)'
func growCI(L *LuaState) int {
	if L.sizeCi > LUAI_MAXCCALLS { /* overflow while handling overflow? */
		L.dThrow(LUA_ERRERR)
	} else {
		L.dReallocCI(2 * L.sizeCi)
		if L.sizeCi > LUAI_MAXCCALLS {
			L.gRunError("stack overflow")
		}
	}
	L.ci++
	return L.ci
}

// 对应C函数：`void luaD_reallocCI (lua_State *L, int newsize)'
func (L *LuaState) dReallocCI(newSize int) {
	oldci := L.baseCi
	L.baseCi = make([]CallInfo, newSize)
	copy(L.baseCi, oldci)
	L.sizeCi = newSize
	L.endCi = L.sizeCi - 1
}
