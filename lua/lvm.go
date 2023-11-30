package golua

import (
	"bytes"
	"errors"
	"fmt"
	"unsafe"
)

const MAXTAGLOOP = 100 /* limit for table tag-method chains (to avoid loops) */

// 对应C函数：`tonumber(o,n)'
func tonumber(o *TValue, n *TValue) bool {
	if o.IsNumber() {
		return true
	}
	o = vToNumber(o, n)
	return o != nil
}

// 对应C函数：`const TValue *luaV_tonumber (const TValue *obj, TValue *n)'
func vToNumber(obj *TValue, n *TValue) *TValue {
	var num LuaNumber
	if obj.IsNumber() {
		return obj
	}
	if obj.IsString() && oStr2d(string(obj.StringValue().Bytes), &num) {
		n.SetNumber(num)
		return n
	} else {
		return nil
	}
}

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
					L.DbgRunError("string length overflow")
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
			top.Ptr(-n).SetString(L, L.sNewStr(buffer[:tl]))
		}
		total -= n - 1 /* got 'n' strings to create 1 new */
		last -= n - 1
	}
}

// 对应C函数：`static void Arith (lua_State *L, StkId ra, const TValue *rb,
//
//	const TValue *rc, TMS op)'
func arith(L *LuaState, ra StkId, rb, rc *TValue, op TMS) {
	b := vToNumber(rb, &TValue{})
	c := vToNumber(rb, &TValue{})
	if b != nil && c != nil {
		nb, nc := b.NumberValue(), c.NumberValue()
		switch op {
		case TM_ADD:
			ra.SetNumber(luai_numadd(nb, nc))
		case TM_SUB:
			ra.SetNumber(luai_numsub(nb, nc))
		case TM_MUL:
			ra.SetNumber(luai_nummul(nb, nc))
		case TM_DIV:
			ra.SetNumber(luai_numdiv(nb, nc))
		case TM_MOD:
			ra.SetNumber(luai_nummod(nb, nc))
		case TM_POW:
			ra.SetNumber(luai_numpow(nb, nc))
		case TM_UNM:
			ra.SetNumber(luai_numunm(nb))
		default:
			LuaAssert(false)
		}
	} else if !callBinTM(L, rb, rc, ra, op) {
		L.gArithError(rb, rc)
	}
}

func toString(L *LuaState, obj StkId) bool {
	return obj.IsString() || obj.vToString(L)
}

// 对应C函数：`void luaV_settable (lua_State *L, const TValue *t, TValue *key, StkId val)'
func (L *LuaState) vSetTable(t *TValue, key *TValue, val StkId) {
	for loop := 0; loop < MAXTAGLOOP; loop++ {
		var tm *TValue
		if t.IsTable() { /* `t' is a table? */
			h := t.TableValue()
			oldVal := h.Set(L, key) /* do a primitive set */
			if !oldVal.IsNil() {
				oldVal.SetObj(L, val)
				L.cBarrierT(h, val)
				return
			}
			if tm = FastTM(L, h.metatable, TM_NEWINDEX); tm == nil { /* or no TM? */
				oldVal.SetObj(L, val)
				L.cBarrierT(h, val)
				return
			}
			/* else will try the tag method */
		} else if tm = L.tGetTMByObj(t, TM_NEWINDEX); tm == nil {
			L.gTypeError(t, "index")
		}

		if tm.IsFunction() {
			callTM(L, tm, t, key, val)
			return
		}
		t = tm /* else repeat with `tm' */
	}
	L.DbgRunError("loop in settable")
}

// 对应C函数：`static int call_binTM (lua_State *L, const TValue *p1, const TValue *p2, StkId res, TMS event)'
func callBinTM(L *LuaState, p1 *TValue, p2 *TValue, res StkId, event TMS) bool {
	// todo: callBinTM
	return false
}

// 对应C函数：`static void callTMres (lua_State *L, StkId res, const TValue *f,
//
//	const TValue *p1, const TValue *p2)'
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

// 对应C函数：`static void callTM (lua_State *L, const TValue *f, const TValue *p1,
//
//	const TValue *p2, const TValue *p3)'
func callTM(L *LuaState, f, p1, p2, p3 *TValue) {
	L.AtTop(0).SetObj(L, f)  /* push function */
	L.AtTop(1).SetObj(L, p1) /* 1st argument */
	L.AtTop(2).SetObj(L, p2) /* 2nd argument */
	L.AtTop(3).SetObj(L, p3) /* 3rd argument */
	L.dCheckStack(4)
	L.top += 4
	L.dCall(L.AtTop(-4), 0)
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
	L.DbgRunError("loop in gettable")
}

// 对应C函数：`void luaV_execute (lua_State *L, int nexeccalls)'
func (L *LuaState) vExecute(nExecCalls int) {
	const (
		ColorCode    = "\u001B[34m"
		ColorReset   = "\u001B[0m"
		ColorSlave   = "\u001B[35m"
		ColorIgnored = "\u001B[36m"
	)
	if DEBUG {
		var k = L.CI().Func().L().p.k
		fmt.Println("\u001B[34mCONSTANTS======================={\u001B[0m")
		for i, value := range k {
			// fmt.Printf("[%d]\t", i)
			var s string
			if value.IsNumber() {
				s = fmt.Sprintf("number: %v", value.NumberValue())
			} else if value.IsString() {
				s = fmt.Sprintf("string: '%s'", string(value.StringValue().GetStr()))
			} else if value.IsFunction() {
				var f = value.ClosureValue()
				if f.IsCFunction() {
					s = fmt.Sprintf("go-func: %p", f.C().f)
				} else {
					s = fmt.Sprintf("lua-func: %p", f.L().p)
				}
			} else {
				s = fmt.Sprintf("%s", L.TypeName(value.gcType()))
			}
			fmt.Printf("\u001B[34m[%d]\t%s\u001B[0m\n", i, s)
		}
		fmt.Print("\u001B[34mCONSTANTS=======================}\u001B[0m\n\n")
	}
reentry: /* entry point */
	LuaAssert(L.CI().IsLua())
	var (
		pc   = L.savedPc
		cl   = L.CI().Func().L()
		base = L.base
		k    = cl.p.k
	)

	var (
		RA = func(i Instruction) *TValue {
			return &L.stack[base+i.GetArgA()]
		}
		RB = func(i Instruction) *TValue {
			CheckExp(getBMode(i.GetOpCode()) == OpArgR)
			return &L.stack[base+i.GetArgB()]
		}
		/*	RC = func(i Instruction) *TValue {
			CheckExp(getCMode(i.GetOpCode()) == OpArgR)
			return &L.stack[base+i.GetArgC()]
		}*/
		RKB = func(i Instruction) *TValue {
			CheckExp(getBMode(i.GetOpCode()) == OpArgK)
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

		getKst = func(i int) string { // DEBUG 使用的辅助函数
			var v = k[i]
			var s string
			if v.IsNumber() {
				s = fmt.Sprintf("<%g>", v.NumberValue())
			} else if v.IsString() {
				s = fmt.Sprintf("<'%s'>", v.StringValue().GetStr())
			} else {
				s = fmt.Sprintf("<%s>", L.TypeName(v.gcType()))
			}
			return "\u001B[31m" + s + "\u001B[34m"
		}
		incrPC = func() {
			pc = pc.Ptr(1)
		}
		dumpCode = func(instruction *Instruction, color string) {
			if DEBUG {
				fmt.Printf("%s%s%s\n",
					color, instruction.DumpCode(getKst, L.top-L.base), ColorReset)
			}
		}
		DoJump = func(n int) {
			pc = pc.Ptr(n)
			L.iThreadYield()
		}
		dumpJumped = func(p *Instruction, n int) {
			if DEBUG {
				for j := 0; j < n; j++ {
					dumpCode(p.Ptr(j), ColorIgnored)
				}
			}
		}
	)

	/* main loop of interpreter */
	for {
		var i = *pc
		pc = pc.Ptr(1) // pc++

		if DEBUG {
			fmt.Printf("\u001B[34m%s\u001B[0m\n", i.DumpCode(getKst, L.top-L.base))
		}

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
		rai := base + i.GetArgA()
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
				dumpCode(pc, ColorIgnored)
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
			L.savedPc = pc // Protect
			L.vGetTable(&g, rb, ra)
			base = L.base
			continue
		case OP_GETTABLE:
			L.savedPc = pc // Protect
			L.vGetTable(RB(i), RKC(i), ra)
			base = L.base
			continue
		case OP_SETGLOBAL:
			var g TValue
			g.SetTable(L, cl.env)
			kbx := KBx(i)
			LuaAssert(kbx.IsString())
			L.savedPc = pc
			L.vSetTable(&g, kbx, ra)
			continue
		case OP_SETUPVAL:
			uv := cl.upVals[i.GetArgB()]
			uv.v.SetObj(L, ra)
			L.cBarrier(uv, ra)
			continue
		case OP_SETTABLE:
			L.savedPc = pc // Protect
			L.vSetTable(ra, RKB(i), RKC(i))
			base = L.base
			continue
		case OP_NEWTABLE:
			b := i.GetArgB()
			c := i.GetArgC()
			ra.SetTable(L, L.hNew(oFb2Int(b), oFb2Int(c)))
			L.savedPc = pc // Protect
			L.cCheckGC()
			base = L.base
			continue
		case OP_SELF:
			var rb = RB(i)
			ra.Ptr(1).SetObj(L, rb)
			L.savedPc = pc // Protect
			L.vGetTable(rb, RKC(i), ra)
			base = L.base
		case OP_ADD:
			var rb = RKB(i)
			var rc = RKC(i)
			if rb.IsNumber() && rc.IsNumber() {
				var nb, nc = rb.NumberValue(), rc.NumberValue()
				ra.SetNumber(luai_numadd(nb, nc))
			} else {
				L.savedPc = pc // Protect
				arith(L, ra, rb, rc, TM_ADD)
				base = L.base
			}
			continue
		case OP_SUB:
			var rb = RKB(i)
			var rc = RKC(i)
			if rb.IsNumber() && rc.IsNumber() {
				var nb, nc = rb.NumberValue(), rc.NumberValue()
				ra.SetNumber(luai_numsub(nb, nc))
			} else {
				L.savedPc = pc // Protect
				arith(L, ra, rb, rc, TM_SUB)
				base = L.base
			}
			continue
		case OP_MUL:
			var rb = RKB(i)
			var rc = RKC(i)
			if rb.IsNumber() && rc.IsNumber() {
				var nb, nc = rb.NumberValue(), rc.NumberValue()
				ra.SetNumber(luai_nummul(nb, nc))
			} else {
				L.savedPc = pc // Protect
				arith(L, ra, rb, rc, TM_MUL)
				base = L.base
			}
			continue
		case OP_DIV:
			var rb = RKB(i)
			var rc = RKC(i)
			if rb.IsNumber() && rc.IsNumber() {
				var nb, nc = rb.NumberValue(), rc.NumberValue()
				ra.SetNumber(luai_numdiv(nb, nc))
			} else {
				L.savedPc = pc // Protect
				arith(L, ra, rb, rc, TM_DIV)
				base = L.base
			}
			continue
		case OP_MOD:
			var rb = RKB(i)
			var rc = RKC(i)
			if rb.IsNumber() && rc.IsNumber() {
				var nb, nc = rb.NumberValue(), rc.NumberValue()
				ra.SetNumber(luai_nummod(nb, nc))
			} else {
				L.savedPc = pc // Protect
				arith(L, ra, rb, rc, TM_MOD)
				base = L.base
			}
			continue
		case OP_UNM:
			var rb = RB(i)
			if rb.IsNumber() {
				var nb = rb.NumberValue()
				ra.SetNumber(luai_numunm(nb))
			} else {
				L.savedPc = pc // Protect
				arith(L, ra, rb, rb, TM_UNM)
				base = L.base
			}
			continue
		case OP_NOT:
			var res = RB(i).IsFalse() /* next assignment may chage this value */
			ra.SetBoolean(res)
			continue
		case OP_LEN:
			var rb = RB(i)
			switch rb.gcType() {
			case LUA_TTABLE:
				ra.SetNumber(LuaNumber(rb.TableValue().GetN()))
			case LUA_TSTRING:
				ra.SetNumber(LuaNumber(rb.StringValue().Len))
			default: /* try metamethod */
				L.savedPc = pc // Protect
				if !callBinTM(L, rb, LuaObjNil, ra, TM_LEN) {
					L.gTypeError(rb, "get length of")
				}
				base = L.base
			}
			continue
		case OP_CONCAT:
			var b = i.GetArgB()
			var c = i.GetArgC()
			L.savedPc = pc // Protect
			L.vConcat(c-b+1, c)
			L.cCheckGC()
			base = L.base
			RA(i).SetObj(L, &L.stack[base+b])
			continue
		case OP_JMP:
			if DEBUG {
				for j := 0; j < i.GetArgSBx(); j++ {
					dumpCode(pc.Ptr(j), ColorIgnored)
				}
			}
			DoJump(i.GetArgSBx())
			continue
		case OP_EQ:
			var rb = RKB(i)
			var rc = RKC(i)
			L.savedPc = pc // Protect
			if equalobj(L, rb, rc) == (i.GetArgA() != 0) {
				DoJump(pc.GetArgSBx())
			}
			base = L.base
			incrPC() // pc++
			continue
		case OP_LT:
			L.savedPc = pc // Protect
			if L.vLessThan(RKB(i), RKC(i)) == (i.GetArgA() != 0) {
				DoJump(pc.GetArgSBx())
			}
			base = L.base
			pc = pc.Ptr(1) // pc++
			continue
		case OP_LE:
			L.savedPc = pc // Protect
			if lessequal(L, RKB(i), RKC(i)) == (i.GetArgA() != 0) {
				DoJump(pc.GetArgSBx())
			}
			base = L.base
			incrPC() // pc++
			continue
		case OP_TEST:
			dumpCode(pc, ColorSlave)
			if ra.IsFalse() != (i.GetArgC() != 0) {
				dumpJumped(pc.Ptr(1), pc.GetArgSBx())
				DoJump(pc.GetArgSBx())
			}
			incrPC() // pc++
			continue
		case OP_TESTSET:
			var rb = RB(i)
			dumpCode(pc, ColorSlave)
			if rb.IsFalse() != (i.GetArgC() != 0) {
				ra.SetObj(L, rb)
				dumpJumped(pc.Ptr(1), pc.GetArgSBx())
				DoJump(pc.GetArgSBx())
			}
			incrPC()
			continue
		case OP_CALL:
			var b = i.GetArgB()            /* 调用函数的参数数量+1，如果b为0则表示函数之上到栈顶都是参数 */
			var nResults = i.GetArgC() - 1 /* 期望的返回值数量，返回值数量不匹配时，在`dPoscall'中进行调整 */
			if b != 0 {
				L.top = base + i.GetArgA() + b /* b=参数个数+1 */
			} /* else previous instruction set top */
			L.savedPc = pc
			switch L.dPrecall(ra, nResults) {
			case PCRLUA:
				nExecCalls++
				goto reentry /* restart luaV_execute over new Lua function */
			case PCRC:
				/* it was a C function (`precall' called it); adjust results */
				if nResults >= 0 {
					L.top = L.CI().top
				}
				base = L.base
				continue
			default:
				return /* yield */
			}
		case OP_TAILCALL:
			var b = i.GetArgB()
			if b != 0 {
				L.top = rai + b
			} /* else previous instruction set top */
			L.savedPc = pc
			LuaAssert(i.GetArgC()-1 == LUA_MULTRET)
			switch L.dPrecall(ra, LUA_MULTRET) {
			case PCRLUA:
				/* tail call: put new frame in place of previous one */
				var ci = L.baseCi[L.ci-1] /* previous frame */
				var fn = ci.fn
				var pfn = L.CI().fn /* previous function index */
				var pfnIdx = adr2idx(L, L.CI().fn)
				var aux int

				if L.openUpval != nil {
					L.fClose(&L.stack[ci.base])
				}
				for aux = 0; pfnIdx+aux < L.top; aux++ {
					fn.Ptr(aux).SetObj(L, pfn.Ptr(aux))
				}
				ci.top = adr2idx(L, fn) + aux /* correct top*/
				L.top = ci.top
				LuaAssert(L.top == L.base+int(fn.LFuncValue().p.maxStackSize))
				ci.savedPc = L.savedPc
				ci.tailCalls++ /* one more call lost */
				L.ci--         /*remove new frame */
				goto reentry
			case PCRC: /* it was a C function (`precall' called it) */
				base = L.base
				continue
			default:
				return /* yield */
			}
		case OP_RETURN:
			var b = i.GetArgB()
			if b != 0 {
				L.top = rai + b - 1
			}
			if L.openUpval != nil {
				L.fClose(&L.stack[base])
			}
			L.savedPc = pc
			b = L.dPoscall(rai)
			nExecCalls--
			if nExecCalls == 0 { /* was previous function running `here'? */
				return /* not: return */
			} else { /* yes: continue its execution */
				if b != 0 {
					L.top = L.CI().top
				}
				LuaAssert(L.CI().IsLua())
				LuaAssert(L.CI().savedPc.Ptr(-1).GetOpCode() == OP_CALL)
				goto reentry
			}
		case OP_FORLOOP:
			var step = ra.Ptr(2).NumberValue()
			var idx = luai_numadd(ra.NumberValue(), step) /* increment index */
			var limit = ra.Ptr(1).NumberValue()
			if incr := luai_numlt(0, step); (incr && luai_numle(idx, limit)) ||
				(!incr && luai_numle(limit, idx)) {
				DoJump(i.GetArgSBx())    /* jump back */
				ra.SetNumber(idx)        /* update internal index... */
				ra.Ptr(3).SetNumber(idx) /* ...and external index */
			}
			continue
		case OP_FORPREP:
			var init = RA(i)
			var pLimit = init.Ptr(1)
			var pStep = init.Ptr(2)
			L.savedPc = pc /* next steps may throw errors */
			if !tonumber(init, RA(i)) {
				L.DbgRunError("'for' initial value must be a number")
			} else if !tonumber(pLimit, RA(i).Ptr(1)) {
				L.DbgRunError("'for' limit must be number")
			} else if !tonumber(pStep, RA(i).Ptr(2)) {
				L.DbgRunError("'for' step must be a number")
			}
			RA(i).SetNumber(luai_numsub(RA(i).NumberValue(), pStep.NumberValue()))
			dumpJumped(pc, i.GetArgSBx())
			DoJump(i.GetArgSBx())
			continue
		case OP_TFORLOOP:
			var cb = ra.Ptr(3) /* call base */
			var cbi = rai + 3  /* call base index */
			cb.Ptr(2).SetObj(L, ra.Ptr(2))
			cb.Ptr(1).SetObj(L, ra.Ptr(1))
			cb.SetObj(L, ra)
			L.top = cbi + 3 /* func. + 2 args (state and index) */
			L.savedPc = pc  // Protect
			L.dCall(cb, i.GetArgC())
			base = L.base
			L.top = L.CI().top
			cb = RA(i).Ptr(3) /* previous call may change the stack */
			if !cb.IsNil() {  /* continue loop? */
				cb.Ptr(-1).SetObj(L, cb) /* save control variable */
				DoJump(pc.GetArgSBx())   /* jump back */
			}
			pc = pc.Ptr(1)
			continue
		case OP_SETLIST:
			var n = i.GetArgB()
			var c = i.GetArgC()
			if n == 0 {
				n = L.top - rai - 1
				L.top = L.CI().top
			}
			if c == 0 {
				c = int(*pc)
				pc = pc.Ptr(1) // pc++
			}
			if !ra.IsTable() { // runtime_check
				break
			}
			var h = ra.TableValue()
			var last = (c-1)*LFIELDS_PER_FLUSH + n
			if last > h.sizeArray { /* needs more space? */
				h.ResizeArray(L, last) /* pre-alloc it at once */
			}
			for ; n > 0; n-- {
				var val = ra.Ptr(n)
				h.SetByNum(L, last).SetObj(L, val)
				last--
				L.cBarrierT(h, val)
			}
			continue
		case OP_CLOSE:
			L.fClose(ra)
			continue
		case OP_CLOSURE:
			var p = cl.p.p[i.GetArgBx()]
			var nup = p.nUps
			var ncl = L.fNewLClosure(nup, cl.env)
			ncl.p = p
			for j := 0; j < nup; j++ {
				if pc.GetOpCode() == OP_GETUPVAL {
					ncl.upVals[j] = cl.upVals[pc.GetArgB()]
				} else {
					LuaAssert(pc.GetOpCode() == OP_MOVE)
					ncl.upVals[j] = L.fFindUpVal(&L.stack[base+pc.GetArgB()])
				}
				if DEBUG {
					fmt.Printf("\u001B[34m%s \u001B[35m", pc.DumpCode(getKst, L.top-L.base)[:33])
					if pc.GetOpCode() == OP_GETUPVAL {
						fmt.Printf("r%d.upvals[%d] := cl.upvals[%d]", i.GetArgA(), j, pc.GetArgB())
					} else {
						fmt.Printf("r%d.upvals[%d] := findupval(r%d)", i.GetArgA(), j, pc.GetArgB())
					}
					fmt.Printf("\u001B[0m\n")
				}
				pc = pc.Ptr(1) // pc++
			}
			ra.SetClosure(L, ncl)
			L.savedPc = pc // Protect
			L.cCheckGC()
			base = L.base
			continue
		case OP_VARARG:
			var b = i.GetArgB() - 1
			var ci = L.CI()
			var n = ci.base - adr2idx(L, ci.fn) - cl.p.numParams - 1
			if b == LUA_MULTRET {
				L.savedPc = pc // Protect
				L.dCheckStack(n)
				base = L.base
				ra = RA(i) /* previous call may change the stack */
				b = n
				L.top = rai + n
			}
			for j := 0; j < b; j++ {
				if j < n {
					ra.Ptr(j).SetObj(L, &L.stack[ci.base-n+j])
				} else {
					ra.Ptr(j).SetNil()
				}
			}
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

// 对应C函数：`equalobj(L,o1,o2)'
func equalobj(L *LuaState, o1, o2 *TValue) bool {
	return o1.gcType() == o2.gcType() && vEqualVal(L, o1, o2)
}

// 对应C函数：`int luaV_equalval (lua_State *L, const TValue *t1, const TValue *t2)'
func vEqualVal(L *LuaState, t1, t2 *TValue) bool {
	var tm *TValue
	LuaAssert(t1.gcType() == t2.gcType())
	switch t1.gcType() {
	case LUA_TNIL:
		return true
	case LUA_TNUMBER:
		return luai_numeq(t1.NumberValue(), t2.NumberValue())
	case LUA_TBOOLEAN:
		return t1.BooleanValue() == t2.BooleanValue() /* true must be 1 !!*/
	case LUA_TLIGHTUSERDATA:
		return t1.PointerValue() == t2.PointerValue()
	case LUA_TUSERDATA:
		if t1.UdataValue() == t2.UdataValue() {
			return true
		}
		tm = get_compTM(L, t1.UdataValue().metatable, t1.UdataValue().metatable, TM_EQ)
		break /* will try TM */
	case LUA_TTABLE:
		if t1.TableValue() == t2.TableValue() {
			return true
		}
		tm = get_compTM(L, t1.TableValue().metatable, t2.TableValue().metatable, TM_EQ)
		break /* will try TM */
	default:
		return t1.GcValue() == t2.GcValue()
	}
	if tm == nil { /* no TM? */
		return false
	}
	callTMRes(L, L.Top(), tm, t1, t2) /* call TM */
	return !L.Top().IsFalse()
}

// 对应C函数：
// static const TValue *get_compTM (lua_State *L, Table *mt1, Table *mt2,
//
//	TMS event)
func get_compTM(L *LuaState, mt1, mt2 *Table, event TMS) *TValue {
	var tm1 = FastTM(L, mt1, event)
	if mt1 == nil { /* no metamethod */
		return nil
	}
	if mt1 == mt2 { /* same metatables => same metamethods */
		return tm1
	}
	var tm2 = FastTM(L, mt2, event)
	if tm2 == nil { /* no metamethod */
		return nil
	}
	if oRawEqualObj(tm1, tm2) { /* same metamethods? */
		return tm1
	}
	return nil
}

// 对应C函数：`int luaV_lessthan (lua_State *L, const TValue *l, const TValue *r)'
func (L *LuaState) vLessThan(l *TValue, r *TValue) bool {
	if l.gcType() != r.gcType() {
		return L.gOrderError(l, r)
	} else if l.IsNumber() {
		return luai_numlt(l.NumberValue(), r.NumberValue())
	} else if l.IsString() {
		return l_strcmp(l.StringValue(), r.StringValue()) < 0
	} else if res, err := call_orderTM(L, l, r, TM_LT); err == nil {
		return res
	}
	return L.gOrderError(l, r)
}

// 对应C函数：`static int lessequal (lua_State *L, const TValue *l, const TValue *r)'
func lessequal(L *LuaState, l *TValue, r *TValue) bool {
	if l.gcType() != r.gcType() {
		return L.gOrderError(l, r)
	} else if l.IsNumber() {
		return luai_numle(l.NumberValue(), r.NumberValue())
	} else if l.IsString() {
		return l_strcmp(l.StringValue(), r.StringValue()) <= 0
	} else if res, err := call_orderTM(L, l, r, TM_LE); err == nil { /* first try `le' */
		return res
	} else if res, err := call_orderTM(L, r, l, TM_LT); err == nil { /* else try 'lt' */
		return !res
	}
	return L.gOrderError(l, r)
}

// 对应C函数：`static int l_strcmp (const TString *ls, const TString *rs)'
func l_strcmp(ls *TString, rs *TString) int {
	return bytes.Compare(ls.Bytes[:ls.Len], rs.Bytes[:rs.Len])
}

// 对应C函数：
// static int call_orderTM (lua_State *L, const TValue *p1, const TValue *p2, TMS event)
func call_orderTM(L *LuaState, p1 *TValue, p2 *TValue, event TMS) (bool, error) {
	var tm1 = L.tGetTMByObj(p1, event)
	if tm1.IsNil() { /* no metamethod? */
		return false, errors.New("no metamethod")
	}
	var tm2 = L.tGetTMByObj(p2, event)
	if !oRawEqualObj(tm1, tm2) { /* different metamethods? */
		return false, errors.New("different metamethods")
	}
	callTMRes(L, L.Top(), tm1, p1, p2)
	return !L.Top().IsFalse(), nil
}
