package golua

import "unsafe"

const MAXTAGLOOP = 100 /* limit for table tag-method chains (to avoid loops) */

// 对应C函数：`void luaV_concat (lua_State *L, int total, int last)'
func (L *LuaState) vConcat(total int, last int) {

	for total > 1 { /* repeat until only 1 result left */
		top := L.Base().Ptr(last + 1)
		p1 := top.Ptr(-2)
		p2 := top.Ptr(-1)
		var n = 2 /* number of elements handled in this pass (at least 2) */
		if !(p1.IsString() || p1.IsNumber()) || !toString(L, p2) {
			if !callBinTM(L, p1, p2, p1, TM_CONCAT) {
				L.gConcatError(p1, p2)
			}
		} else if p2.StringValue().Len == 0 { /* second op is empty? */
			toString(L, p1) /* result is first op (as string) */
		} else {
			/* at least two string values; get as many as possible */
			tl := top.Ptr(-1).StringValue().Len
			/* collect total length */
			for n = 1; n < total && toString(L, top.Ptr(-n-1)); n++ {
				l := top.Ptr(-n - 1).StringValue().Len
				if l >= int(MAX_SIZET)-tl {
					L.gRunError("string length overflow")
				}
				tl += l
			}
			buffer := L.G().buff.OpenSpace(tl)
			tl = 0
			for i := n; i > 0; i-- { /* concat all strings */
				s := top.Ptr(-i).StringValue()
				copy(buffer[tl:], s.Bytes)
				tl += s.Len
			}
			top.Ptr(-n).SetString(L, L.sNewLStr(buffer[:tl]))
		}
		total -= n - 1 /* got 'n' strings to create 1 new */
		last -= n - 1
	}
}

func toString(L *LuaState, obj StkId) bool {
	return obj.IsString() || obj.vToString(L)
}

// 对应C函数：`static int call_binTM (lua_State *L, const TValue *p1, const TValue *p2, StkId res, TMS event)'
func callBinTM(L *LuaState, p1 *TValue, p2 *TValue, res StkId, event TMS) bool {
	// todo: callBinTM
	return false
}

// 对应C函数：`static void callTMres (lua_State *L, StkId res, const TValue *f,
//                        const TValue *p1, const TValue *p2)'
func callTMRes(L *LuaState, res StkId, f *TValue, p1 *TValue, p2 *TValue) {
	result := savestack(L, res)
	L.Top().SetObj(L, f)     /* push function */
	L.AtTop(1).SetObj(L, p1) /* 1st argument */
	L.AtTop(2).SetObj(L, p2) /* 2nd argument */
	L.dCheckStack(3)
	L.top += 3
	L.dCall(L.AtTop(-3), 1)
	res = restorestack(L, result)
	L.top--
	res.SetObj(L, L.Top())
}

// 对应C函数：`void luaV_gettable (lua_State *L, const TValue *t, TValue *key, StkId val)'
func (L *LuaState) vGetTable(t *TValue, key *TValue, val StkId) {
	for loop := 0; loop < MAXTAGLOOP; loop++ {
		var tm *TValue
		if t.IsTable() { /* `t' is a table? */
			h := t.TableValue()
			res := h.Get(key) /* do a primitive get */
			if !res.IsNil() {
				val.SetObj(L, res)
				return
			}
			if tm = FastTM(L, h.metatable, TM_INDEX); tm == nil { /* or no TM? */
				val.SetObj(L, res)
				return
			}
			/* else will try the tag method */
		} else if tm = L.tGetTMByObj(t, TM_INDEX); tm.IsNil() {
			L.gTypeError(t, "index")
		}
		if tm.IsFunction() {
			callTMRes(L, val, tm, t, key)
			return
		}
		t = tm /* else repreat with `tm' */
	}
	L.gRunError("loop in gettable")
}

// 对应C函数：`void luaV_execute (lua_State *L, int nexeccalls)'
func (L *LuaState) vExecute(nExecCalls int) {

reentry: /* entry point */
	LuaAssert(L.CI().IsLua())
	pc := L.savedPc
	cl := L.CI().Func().L()
	base := L.base
	k := cl.p.k

	var (
		RA = func(i Instruction) *TValue {
			return &L.stack[base+i.GetArgA()]
		}
		RB = func(i Instruction) *TValue {
			CheckExp(getBMode(i.GetOpCode()) == OpArgR)
			return &L.stack[base+i.GetArgB()]
		}
		RC = func(i Instruction) *TValue {
			CheckExp(getCMode(i.GetOpCode()) == OpArgR)
			return &L.stack[base+i.GetArgC()]
		}
		RKB = func(i Instruction) *TValue {
			CheckExp(getBMode(i.GetArgB()) == OpArgK)
			B := i.GetArgB()
			if ISK(B) {
				return &k[INDEXK(B)]
			} else {
				return &L.stack[base+B]
			}
		}
		RKC = func(i Instruction) *TValue {
			CheckExp(getCMode(i.GetOpCode()) == OpArgK)
			C := i.GetArgC()
			if ISK(C) {
				return &k[INDEXK(C)]
			} else {
				return &L.stack[base+C]
			}
		}
		KBx = func(i Instruction) *TValue {
			CheckExp(getBMode(i.GetOpCode()) == OpArgK)
			return &k[i.GetArgBx()]
		}
	)

	/* main loop of interpreter */
	for {
		i := *pc
		pc = pc.Ptr(1) // pc++
		if L.hookMask&(LUA_MASKLINE|LUA_MASKCOUNT) != 0 &&
			(L.DecrHookCount() == 0 || L.hookMask&LUA_MASKLINE != 0) {
			traceexec(L, pc)
			if L.status == LUA_YIELD { /* di hook yield? */
				L.savedPc = pc.Ptr(-1)
				return
			}
			base = L.base
		}
		/* warning!! several calls may realloc the stack and invalidate `ra' */
		ra := RA(i)
		LuaAssert(base == L.base && L.base == L.CI().base)
		LuaAssert(base <= L.top && L.top <= L.stackSize)
		LuaAssert(L.top == L.CI().top || gCheckOpenOp(i))
		switch i.GetOpCode() {
		case OP_MOVE:
			ra.SetObj(L, RB(i))
			continue
		case OP_LOADK:
			ra.SetObj(L, KBx(i))
			continue
		case OP_LOADBOOL:
			ra.SetBoolean(i.GetArgB() == 1)
			if i.GetArgC() != 0 { /* skip next instruction (if C) */
				pc = pc.Ptr(1) // pc++
			}
			continue
		case OP_LOADNIL:
			rb := RB(i)
			for adr2idx(L, rb) >= adr2idx(L, ra) {
				rb.SetNil()
				rb = rb.Ptr(-1) // rb--
			}
			continue
		case OP_GETUPVAL:
			b := i.GetArgB()
			ra.SetObj(L, cl.upVals[b].v)
			continue
		case OP_GETGLOBAL:
			var g TValue
			rb := KBx(i)
			g.SetTable(L, cl.env)
			LuaAssert(rb.IsString())
			// Protect
			L.savedPc = pc
			L.vGetTable(&g, rb, ra)
			base = L.base
			continue
		}
	}
}

// 对应C函数：`static void traceexec (lua_State *L, const Instruction *pc)'
func traceexec(L *LuaState, pc *Instruction) {
	mask := L.hookMask
	oldPc := L.savedPc
	L.savedPc = pc
	if (mask&LUA_MASKCOUNT != 0) && L.hookCount == 0 {
		ResetHookCount(L)
		L.dCallHook(LUA_HOOKCOUNT, -1)
	}
	if mask&LUA_MASKLINE != 0 {
		p := L.CI().Func().L().p
		npc := p.pcRel(pc)
		newLine := p.getLine(npc)
		/* call linehook when enter a new function, when jump back (loop),
		   or when enter a new line */
		if npc == 0 || uintptr(unsafe.Pointer(pc)) <= uintptr(unsafe.Pointer(oldPc)) ||
			newLine != p.getLine(p.pcRel(oldPc)) {
			L.dCallHook(LUA_HOOKLINE, newLine)
		}
	}
}
