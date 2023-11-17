package golua

import "math"

// NO_JUMP
// Marks the end of patch list. It is an invalid value both as an absolute
// address, and as a list link (would link an element to itself).
const NO_JUMP = -1

type BinOpr int /* 对应C类型：`enum BinOpr' */

// grep "ORDER OPR" if you change these enums
const (
	OPR_ADD BinOpr = iota
	OPR_SUB
	OPR_MUL
	OPR_DIV
	OPR_MOD
	OPR_POW
	OPR_CONCAT
	OPR_NE
	OPR_EQ
	OPR_LT
	OPR_LE
	OPR_GT
	OPR_GE
	OPR_AND
	OPR_OR
	OPR_NOBINOPR
)

type UnOpr = int /* 对应C类型：`enum UnOpr' */
const (
	OPR_MINUS UnOpr = iota
	OPR_NOT
	OPR_LEN
	OPR_NOUNOPR
)

// 对应C函数：`void luaK_prefix (FuncState *fs, UnOpr op, expdesc *e)'
func (fs *FuncState) kPrefix(op UnOpr, e *expdesc) {
	var e2 = &expdesc{
		k:    VKNUM,
		nval: 0,
		t:    NO_JUMP,
		f:    NO_JUMP,
	}
	switch op {
	case OPR_MINUS:
		if !e.isNumeral() {
			fs.kExp2anyReg(e) /* cannot operate on non-numeric constants */
		}
		fs.codeArith(OP_UNM, e, e2)
	case OPR_NOT:
		fs.codeNot(e)
	case OPR_LEN:
		fs.kExp2anyReg(e) /* cannot operate on constants */
		fs.codeArith(OP_LEN, e, e2)
	default:
		LuaAssert(false)
	}
}

// 对应C函数：`hasjumps(e)'
func (e *expdesc) hasJumps() bool {
	return e.t != e.f
}

// 对应C函数：`static int isnumeral(expdesc *e)'
func (e *expdesc) isNumeral() bool {
	return e.k == VKNUM && e.t == NO_JUMP && e.f == NO_JUMP
}

// 对应C函数：`static void init_exp (expdesc *e, expkind k, int i)'
func (e *expdesc) initExp(k expkind, i int) {
	e.f = NO_JUMP
	e.t = NO_JUMP
	e.k = k
	e.s.info = i
}

// 对应C函数：`int luaK_exp2anyreg (FuncState *fs, expdesc *e)'
func (fs *FuncState) kExp2anyReg(e *expdesc) int {
	fs.kDischargeVars(e)
	if e.k == VNONRELOC {
		if !e.hasJumps() {
			return e.s.info /* exp is already in a register */
		}
		if e.s.info >= int(fs.nActVar) { /* reg. is not a local? */
			fs.exp2reg(e, e.s.info) /* put value on it */
			return e.s.info
		}
	}
	fs.kExp2NextReg(e) /* default */
	return e.s.info
}

// 对应C函数：`void luaK_exp2nextreg (FuncState *fs, expdesc *e)'
func (fs *FuncState) kExp2NextReg(e *expdesc) {
	fs.kDischargeVars(e)
	fs.freeExp(e)
	fs.kReserveRegs(1)
	fs.exp2reg(e, fs.freeReg-1)
}

// 对应C函数：`void luaK_dischargevars (FuncState *fs, expdesc *e)'
func (fs *FuncState) kDischargeVars(e *expdesc) {
	switch e.k {
	case VLOCAL:
		e.k = VNONRELOC
	case VUPVAL:
		e.s.info = fs.kCodeABC(OP_GETUPVAL, 0, e.s.info, 0)
		e.k = VRELOCABLE
	case VGLOBAL:
		e.s.info = fs.kCodeABx(OP_GETGLOBAL, 0, e.s.info)
		e.k = VRELOCABLE
	case VINDEXED:
		fs.freereg(e.s.aux)
		fs.freereg(e.s.info)
		e.s.info = fs.kCodeABC(OP_GETTABLE, 0, e.s.info, e.s.aux)
		e.k = VRELOCABLE
	case VVARGARG, VCALL:
		fs.kSetOneRet(e)
	default:
		break /* there is one value available (somewhere) */
	}
}

// 对应C函数：`int luaK_codeABC (FuncState *fs, OpCode o, int a, int b, int c)'
func (fs *FuncState) kCodeABC(op OpCode, a int, b int, c int) int {
	LuaAssert(op.getOpMode() == iABC)
	LuaAssert(getBMode(op) != OpArgN || b == 0)
	LuaAssert(getCMode(op) != OpArgN || c == 0)
	return fs.kCode(CreateABC(op, a, b, c), fs.ls.lastLine)
}

// 对应C函数：`int luaK_codeABx (FuncState *fs, OpCode o, int a, unsigned int bc)'
func (fs *FuncState) kCodeABx(op OpCode, a int, bc int) int {
	LuaAssert(op.getOpMode() == iABx || op.getOpMode() == iAsBx)
	LuaAssert(getCMode(op) == OpArgN)
	return fs.kCode(CreateABx(op, a, bc), fs.ls.lastLine)
}

// 对应C函数：`luaK_codeAsBx(fs,o,A,sBx)'
func (fs *FuncState) kCodeAsBx(o OpCode, a int, sBx int) int {
	return fs.kCodeABx(o, a, sBx+MAXARG_sBx)
}

// 对应C函数：`static int luaK_code (FuncState *fs, Instruction i, int line)'
func (fs *FuncState) kCode(i Instruction, line int) int {
	var f = fs.f
	fs.dischargeJpc() /* `pc' will change */
	/* put new instruction in code array */
	mGrowVector(fs.L, &f.code, fs.pc, &f.sizeCode,
		MAX_INT, "code size overflow")
	f.code[fs.pc] = i
	/* save corresponding line information */
	mGrowVector(fs.L, &f.lineInfo, fs.pc, &f.sizeLineInfo,
		MAX_INT, "code size overflow")
	f.lineInfo[fs.pc] = line
	fs.pc++
	return fs.pc - 1
}

// 对应C函数：`static void dischargejpc (FuncState *fs)'
func (fs *FuncState) dischargeJpc() {
	fs.patchListAux(fs.jpc, fs.pc, NO_REG, fs.pc)
	fs.jpc = NO_JUMP
}

// 对应C函数：`static void patchlistaux (FuncState *fs, int list, int vtarget, int reg, int dtarget)'
func (fs *FuncState) patchListAux(list int, vtarget int, reg int, dtarget int) {
	for list != NO_JUMP {
		if fs.patchTestReg(list, reg) {
			fs.fixJump(list, vtarget)
		} else {
			fs.fixJump(list, dtarget) /* jump to default target */
		}
		list = fs.getJump(list)
	}
}

// 对应C函数：`static int getjump (FuncState *fs, int pc)'
func (fs *FuncState) getJump(pc int) int {
	var offset = fs.f.code[pc].GetArgSBx()
	if offset == NO_JUMP { /* point to itself represents end of list */
		return NO_JUMP /* end of list */
	} else {
		return pc + 1 + offset /* turn offset into absolute position */
	}
}

// 对应C函数：`static int patchtestreg (FuncState *fs, int node, int reg)'
func (fs *FuncState) patchTestReg(node int, reg int) bool {
	var i = fs.getJumpControl(node)
	if i.GetOpCode() != OP_TESTSET {
		return false /* cannot patch other instructions */
	}
	if reg != NO_REG && reg != i.GetArgB() {
		i.SetArgA(reg)
	} else { /* no register to put value or register already has the value */
		*i = CreateABC(OP_TEST, i.GetArgB(), 0, i.GetArgC())
	}
	return true
}

// 获取跳转指令前面的条件指令（跳转指令由条指令组成，前面有一个条件，后面是跳转地址）
// 对应C函数：`static Instruction *getjumpcontrol (FuncState *fs, int pc)'
func (fs *FuncState) getJumpControl(pc int) *Instruction {
	var pi = &fs.f.code[pc]
	if pc >= 1 && testTMode(pi.Ptr(-1).GetOpCode()) {
		return pi.Ptr(-1)
	} else {
		return pi
	}
}

// 对应C函数：`static void fixjump (FuncState *fs, int pc, int dest)'
func (fs *FuncState) fixJump(pc int, dest int) {
	var jmp = &fs.f.code[pc]
	var offset = dest - (pc + 1)
	LuaAssert(dest != NO_JUMP)
	if int(math.Abs(float64(offset))) > MAXARG_sBx {
		fs.ls.xSyntaxError("control structure too long")
	}
	jmp.SetArgSBx(offset)
}

// 对应C函数：`static void freereg (FuncState *fs, int reg)'
func (fs *FuncState) freereg(reg int) {
	if !ISK(reg) && reg >= fs.nActVar {
		fs.freeReg--
		LuaAssert(reg == fs.freeReg)
	}
}

// 对应C函数：`static void freeexp (FuncState *fs, expdesc *e)'
func (fs *FuncState) freeExp(e *expdesc) {
	if e.k == VNONRELOC {
		fs.freereg(e.s.info)
	}
}

// 对应C函数：`void luaK_reserveregs (FuncState *fs, int n)'
func (fs *FuncState) kReserveRegs(n int) {
	fs.kCheckStack(n)
	fs.freeReg += n
}

// 对应C函数：`void luaK_checkstack (FuncState *fs, int n)'
func (fs *FuncState) kCheckStack(n int) {
	var newStack = fs.freeReg + n
	if newStack > fs.f.maxStackSize {
		if newStack >= MAXSTACK {
			fs.ls.xSyntaxError("function or expression too complex")
		}
		fs.f.maxStackSize = newStack
	}
}

// 对应C函数：`void luaK_setoneret (FuncState *fs, expdesc *e)'
func (fs *FuncState) kSetOneRet(e *expdesc) {
	if e.k == VCALL { /* expression is an open function call? */
		e.k = VNONRELOC
		e.s.info = fs.getCode(e).GetArgA()
	} else if e.k == VVARGARG {
		fs.getCode(e).SetArgB(2)
		e.k = VRELOCABLE /* can relocate its simple result */
	}
}

// 对应C函数：`getcode(fs,e)'
func (fs *FuncState) getCode(e *expdesc) *Instruction {
	return &fs.f.code[e.s.info]
}

// 对应C函数：`static void exp2reg (FuncState *fs, expdesc *e, int reg)'
func (fs *FuncState) exp2reg(e *expdesc, reg int) {
	fs.discharge2reg(e, reg)
	if e.k == VJMP {
		fs.kConcat(&e.t, e.s.info) /* put this jump in `t' list */
	}
	if e.hasJumps() {
		var p_f = NO_JUMP /* position of an eventual LOAD false */
		var p_t = NO_JUMP /* position of an eventual LOAD true */
		if fs.needValue(e.t) || fs.needValue(e.f) {
			var fj int
			if e.k == VJMP {
				fj = NO_JUMP
			} else {
				fj = fs.kJump()
			}
			p_f = fs.codeLabel(reg, 0, 1)
			p_t = fs.codeLabel(reg, 1, 0)
			fs.kPatchToHere(fj)
		}
		var final = fs.kGetLabel() /* position after whole expression */
		fs.patchListAux(e.f, final, reg, p_f)
		fs.patchListAux(e.t, final, reg, p_t)
	}
	e.f = NO_JUMP
	e.t = NO_JUMP
	e.s.info = reg
	e.k = VNONRELOC
}

// 对应C函数：`static int code_label (FuncState *fs, int A, int b, int jump)'
func (fs *FuncState) codeLabel(A int, b int, jump int) int {
	fs.kGetLabel() /* those instructions may be jump targets */
	return fs.kCodeABC(OP_LOADBOOL, A, b, jump)
}

// 对应C函数：`static void discharge2reg (FuncState *fs, expdesc *e, int reg) '
func (fs *FuncState) discharge2reg(e *expdesc, reg int) {
	fs.kDischargeVars(e)
	switch e.k {
	case VNIL:
		fs.kNil(reg, 1)
	case VTRUE:
		fs.kCodeABC(OP_LOADBOOL, reg, 1, 0)
	case VFALSE:
		fs.kCodeABC(OP_LOADBOOL, reg, 0, 0)
	case VK:
		fs.kCodeABx(OP_LOADK, reg, e.s.info)
	case VKNUM:
		fs.kCodeABx(OP_LOADK, reg, fs.kNumberK(e.nval))
	case VRELOCABLE:
		fs.getCode(e).SetArgA(reg)
	case VNONRELOC:
		if reg != e.s.info {
			fs.kCodeABC(OP_MOVE, reg, e.s.info, 0)
		}
	default:
		LuaAssert(e.k == VVOID || e.k == VJMP)
		return /* nothing to do... */
	}
	e.s.info = reg
	e.k = VNONRELOC
}

// 对应C函数：`void luaK_nil (FuncState *fs, int from, int n)'
func (fs *FuncState) kNil(from int, n int) {
	if fs.pc > fs.lastTarget { /* no jumps to current position? */
		if fs.pc == 0 { /* function start? */
			if from >= fs.nActVar {
				return /* positions are already clean */
			}
		} else {
			var previous = &fs.f.code[fs.pc-1]
			if previous.GetOpCode() == OP_LOADNIL {
				var pfrom = previous.GetArgA()
				var pto = previous.GetArgB()
				if pfrom <= from && from <= pto+1 { /* can connect both? */
					if from+n-1 > pto {
						previous.SetArgB(from + n - 1)
					}
					return
				}
			}
		}
	}
	fs.kCodeABC(OP_LOADNIL, from, from+n-1, 0) /* else no optimization */
}

// 对应C函数：`int luaK_jump (FuncState *fs)'
func (fs *FuncState) kJump() int {
	var jpc = fs.jpc /* save list of jumps to here */
	fs.jpc = NO_JUMP
	var j = fs.kCodeAsBx(OP_JMP, 0, NO_JUMP)
	fs.kConcat(&j, jpc) /* keep them on hold */
	return j
}

// 对应C函数：`int luaK_numberK (FuncState *fs, lua_Number r)'
func (fs *FuncState) kNumberK(r LuaNumber) int {
	var o = &TValue{}
	o.SetNumber(r)
	return fs.addk(o, o)
}

// 添加一个常量，保存在fs.h常量表中
// 对应C函数：`static int addk (FuncState *fs, TValue *k, TValue *v)'
func (fs *FuncState) addk(k *TValue, v *TValue) int {
	var L = fs.L
	var idx = fs.h.Set(L, k)
	var f = fs.f
	var oldSize = f.sizeK
	if idx.IsNumber() {
		LuaAssert(oRawEqualObj(&fs.f.k[int(idx.NumberValue())], v))
		return int(idx.NumberValue())
	} else { /* constant not found; create a new entry */
		idx.SetNumber(LuaNumber(fs.nk))
		mGrowVector(L, &f.k, fs.nk, &f.sizeK, MAXARG_Bx, "constant table overflow")
		for ; oldSize < f.sizeK; oldSize++ {
			f.k[oldSize].SetNil()
		}
		f.k[fs.nk].SetObj(L, v)
		L.cBarrier(f, v)
		fs.nk++
		return fs.nk - 1
	}
}

// 对应C函数：`void luaK_concat (FuncState *fs, int *l1, int l2)'
func (fs *FuncState) kConcat(l1 *int, l2 int) {
	if l2 == NO_JUMP {
		return
	}
	if *l1 == NO_JUMP {
		*l1 = l2
	} else {
		var list = *l1
		for next := fs.getJump(list); next != NO_JUMP; next = fs.getJump(list) {
			list = next /* find last element */
		}
		fs.fixJump(list, l2)
	}
}

// check whether list has any jump that do not produce a value
// (or produce an inverted value)
// 对应C函数：`static int need_value (FuncState *fs, int list)'
func (fs *FuncState) needValue(list int) bool {
	for ; list != NO_JUMP; list = fs.getJump(list) {
		var i = fs.getJumpControl(list)
		if i.GetOpCode() != OP_TESTSET {
			return true
		}
	}
	return false /* not found */
}

// returns current `pc' and marks it as a jump target (to avoid wrong
// optimizations with consecutive instructions not in the same basic block).
// 对应C函数：`int luaK_getlabel (FuncState *fs)'
func (fs *FuncState) kGetLabel() int {
	fs.lastTarget = fs.pc
	return fs.pc
}

// 对应C函数：`void luaK_patchtohere (FuncState *fs, int list)'
func (fs *FuncState) kPatchToHere(list int) {
	fs.kGetLabel()
	fs.kConcat(&fs.jpc, list)
}

// 对应C函数：`static void codearith (FuncState *fs, OpCode op, expdesc *e1, expdesc *e2)'
func (fs *FuncState) codeArith(op OpCode, e1 *expdesc, e2 *expdesc) {
	if constFolding(op, e1, e2) {
		return
	}
	var o2 = 0
	if op != OP_UNM && op != OP_LEN {
		o2 = fs.kExp2RK(e2)
	}
	var o1 = fs.kExp2RK(e1)
	if o1 > o2 {
		fs.freeExp(e1)
		fs.freeExp(e2)
	} else {
		fs.freeExp(e2)
		fs.freeExp(e1)
	}
	e1.s.info = fs.kCodeABC(op, 0, o1, o2)
	e1.k = VRELOCABLE
}

// 对应C函数：`static int constfolding (OpCode op, expdesc *e1, expdesc *e2)'
func constFolding(op OpCode, e1, e2 *expdesc) bool {
	if !e1.isNumeral() || !e2.isNumeral() {
		return false
	}
	var v1 = e1.nval
	var v2 = e2.nval
	var r LuaNumber
	switch op {
	case OP_ADD:
		r = luai_numadd(v1, v2)
	case OP_SUB:
		r = luai_numsub(v1, v2)
	case OP_MUL:
		r = luai_nummul(v1, v2)
	case OP_DIV:
		if v2 == 0 {
			return false /* do not attempt to divide by 0 */
		}
		r = luai_numdiv(v1, v2)
	case OP_MOD:
		if v2 == 0 {
			return false /* do not attempt to divide by 0 */
		}
		r = luai_nummod(v1, v2)
	case OP_POW:
		r = luai_numpow(v1, v2)
	case OP_UNM:
		r = luai_numunm(v1)
	case OP_LEN:
		return false /* no constant folding for `len' */
	default:
		LuaAssert(false)
		r = 0
	}
	if luai_numisnan(r) {
		return false /* do not attempt to produce NaN */
	}
	e1.nval = r
	return true
}

// 对应C函数：`int luaK_exp2RK (FuncState *fs, expdesc *e)'
func (fs *FuncState) kExp2RK(e *expdesc) int {
	fs.kExp2val(e)
	switch e.k {
	case VKNUM, VTRUE, VFALSE, VNIL:
		if fs.nk <= MAXINDEXRK { /* constant fit in RK operand? */
			if e.k == VNIL {
				e.s.info = fs.nilK()
			} else if e.k == VKNUM {
				e.s.info = fs.kNumberK(e.nval)
			} else {
				e.s.info = fs.boolK(e.k == VTRUE)
			}
			e.k = VK
			return RKASK(e.s.info)
		}
	case VK:
		if e.s.info <= MAXINDEXRK { /* constant fit in argC? */
			return RKASK(e.s.info)
		}
	default:
		break
	}
	/* not a constant in the right range: put it in a register */
	return fs.kExp2anyReg(e)
}

// 对应C函数：`void luaK_exp2val (FuncState *fs, expdesc *e)'
func (fs *FuncState) kExp2val(e *expdesc) {
	if e.hasJumps() {
		fs.kExp2anyReg(e)
	} else {
		fs.kDischargeVars(e)
	}
}

// 对应C函数：`static int nilK (FuncState *fs)'
func (fs *FuncState) nilK() int {
	var k, v TValue
	v.SetNil()
	/* cannot use nil as key; instead use table itself to represent nil */
	k.SetTable(fs.L, fs.h)
	return fs.addk(&k, &v)
}

// 对应C函数：`static int boolK (FuncState *fs, int b)'
func (fs *FuncState) boolK(b bool) int {
	var o TValue
	o.SetBoolean(b)
	return fs.addk(&o, &o)
}

// 对应C函数：`static void codenot (FuncState *fs, expdesc *e)'
func (fs *FuncState) codeNot(e *expdesc) {
	fs.kDischargeVars(e)
	switch e.k {
	case VNIL, VFALSE:
		e.k = VTRUE
	case VK, VKNUM, VTRUE:
		e.k = VFALSE
	case VJMP:
		fs.invertJump(e)
	case VRELOCABLE, VNONRELOC:
		fs.discharge2anyReg(e)
		fs.freeExp(e)
		e.s.info = fs.kCodeABC(OP_NOT, 0, e.s.info, 0)
		e.k = VRELOCABLE
	default:
		LuaAssert(false) /* cannot happen */
	}
	/* interchange true and false lists */
	var temp = e.f
	e.f = e.t
	e.t = temp
	fs.removeValues(e.f)
	fs.removeValues(e.t)
}

// 对应C函数：`static void invertjump (FuncState *fs, expdesc *e)'
func (fs *FuncState) invertJump(e *expdesc) {
	var pc = fs.getJumpControl(e.s.info)
	LuaAssert(testTMode(pc.GetOpCode()) && pc.GetOpCode() != OP_TESTSET &&
		pc.GetOpCode() != OP_TEST)
	pc.SetArgA((pc.GetArgA() + 1) % 2) /* 0 <-> 1 */
}

// 对应C函数：`static void discharge2anyreg (FuncState *fs, expdesc *e)'
func (fs *FuncState) discharge2anyReg(e *expdesc) {
	if e.k != VNONRELOC {
		fs.kReserveRegs(1)
		fs.discharge2reg(e, fs.freeReg-1)
	}
}

// 对应C函数：`static void removevalues (FuncState *fs, int list)'
func (fs *FuncState) removeValues(list int) {
	for ; list != NO_JUMP; list = fs.getJump(list) {
		fs.patchTestReg(list, NO_REG)
	}
}

// 对应C函数：`int luaK_stringK (FuncState *fs, TString *s)'
func (fs *FuncState) kStringK(s *TString) int {
	var o = &TValue{}
	o.SetString(fs.L, s)
	return fs.addk(o, o)
}

// 对应C函数：`void luaK_setlist (FuncState *fs, int base, int nelems, int tostore)'
func (fs *FuncState) kSetList(base int, nElems int, tostore int) {
	var c = (nElems-1)/LFIELDS_PER_FLUSH + 1
	var b = tostore
	if tostore == LUA_MULTRET {
		b = 0
	}
	LuaAssert(tostore != 0)
	if c <= MAXARG_C {
		fs.kCodeABC(OP_SETLIST, base, b, c)
	} else {
		fs.kCodeABC(OP_SETLIST, base, b, 0)
		fs.kCode(Instruction(c), fs.ls.lastLine)
	}
	fs.freeReg = base + 1 /* free registers with list values */
}

// 对应C函数：`luaK_setmultret(fs,e)'
func (fs *FuncState) kSetMultRet(e *expdesc) {
	fs.kSetReturns(e, LUA_MULTRET)
}

// 对应C函数：`void luaK_setreturns (FuncState *fs, expdesc *e, int nresults)'
func (fs *FuncState) kSetReturns(e *expdesc, nResults int) {
	if e.k == VCALL { /* expression is an open function call? */
		fs.getCode(e).SetArgC(nResults + 1)
	} else if e.k == VVARGARG {
		fs.getCode(e).SetArgB(nResults + 1)
		fs.getCode(e).SetArgA(fs.freeReg)
		fs.kReserveRegs(1)
	}
}

// 对应C函数：`void luaK_ret (FuncState *fs, int first, int nret)'
func (fs *FuncState) kRet(first int, nRet int) {
	fs.kCodeABC(OP_RETURN, first, nRet+1, 0)
}

// 对应C函数：`void luaK_self (FuncState *fs, expdesc *e, expdesc *key)'
func (fs *FuncState) kSelf(e *expdesc, key *expdesc) {
	fs.kExp2anyReg(e)
	fs.freeExp(e)
	var fn = fs.freeReg
	fs.kReserveRegs(2)
	fs.kCodeABC(OP_SELF, fn, e.s.info, fs.kExp2RK(key))
	fs.freeExp(key)
	e.s.info = fn
	e.k = VNONRELOC
}

// 对应C函数：`void luaK_fixline (FuncState *fs, int line)'
func (fs *FuncState) kFixLine(line int) {
	fs.f.lineInfo[fs.pc-1] = line
}

// 对应C函数：`void luaK_infix (FuncState *fs, BinOpr op, expdesc *v)'
func (fs *FuncState) kInfix(op BinOpr, v *expdesc) {
	switch op {
	case OPR_AND:
		fs.kGoIfTrue(v)
	case OPR_OR:
		fs.kGoIfFalse(v)
	case OPR_CONCAT:
		fs.kExp2NextReg(v) /* operand must be on the `stack' */
	case OPR_ADD, OPR_SUB, OPR_MUL, OPR_DIV, OPR_MOD, OPR_POW:
		if !v.isNumeral() {
			fs.kExp2RK(v)
		}
	default:
		fs.kExp2RK(v)
	}
}

// 对应C函数：`void luaK_goiftrue (FuncState *fs, expdesc *e)'
func (fs *FuncState) kGoIfTrue(e *expdesc) {
	var pc int /* pc of last jump */
	fs.kDischargeVars(e)
	switch e.k {
	case VK, VKNUM, VTRUE:
		pc = NO_JUMP /* always true; do nothing */
	case VFALSE:
		pc = fs.kJump() /* always jump */
	case VJMP:
		fs.invertJump(e)
		pc = e.s.info
	default:
		pc = fs.jumpOnCond(e, 0)
	}
	fs.kConcat(&e.f, pc) /* insert last jump in `f' list */
	fs.kPatchToHere(e.t)
	e.t = NO_JUMP
}

// 对应C函数：`static int jumponcond (FuncState *fs, expdesc *e, int cond)'
func (fs *FuncState) jumpOnCond(e *expdesc, cond int) int {
	if e.k == VRELOCABLE {
		var ie = fs.getCode(e)
		if ie.GetOpCode() == OP_NOT {
			fs.pc-- /* remove previous OP_NOT */
			return fs.condJump(OP_TEST, ie.GetArgB(), 0, (cond+1)%2)
		}
		/* else go through */
	}
	fs.discharge2anyReg(e)
	fs.freeExp(e)
	return fs.condJump(OP_TESTSET, NO_REG, e.s.info, cond)
}

// 对应C函数：`static int condjump (FuncState *fs, OpCode op, int A, int B, int C)'
func (fs *FuncState) condJump(op OpCode, A int, B int, C int) int {
	fs.kCodeABC(op, A, B, C)
	return fs.kJump()
}

// 对应C函数：`static void luaK_goiffalse (FuncState *fs, expdesc *e)'
func (fs *FuncState) kGoIfFalse(e *expdesc) {
	var pc int /* pc of last jump */
	fs.kDischargeVars(e)
	switch e.k {
	case VNIL, VFALSE:
		pc = NO_JUMP /* always false; do nothing */
	case VTRUE:
		pc = fs.kJump() /* always jump */
	case VJMP:
		pc = e.s.info
	default:
		pc = fs.jumpOnCond(e, 1)
	}
	fs.kConcat(&e.t, pc) /* insert last jump in `t' list */
	fs.kPatchToHere(e.f)
	e.f = NO_JUMP
}

// 对应C函数：`void luaK_posfix (FuncState *fs, BinOpr op, expdesc *e1, expdesc *e2)'
func (fs *FuncState) kPosFix(op BinOpr, e1 *expdesc, e2 *expdesc) {
	switch op {
	case OPR_AND:
		LuaAssert(e1.t == NO_JUMP) /* list must be closed */
		fs.kDischargeVars(e2)
		fs.kConcat(&e2.f, e1.f)
		*e1 = *e2
	case OPR_OR:
		LuaAssert(e1.f == NO_JUMP) /* list must be closed */
		fs.kDischargeVars(e2)
		fs.kConcat(&e2.t, e1.t)
		*e1 = *e2
	case OPR_CONCAT:
		fs.kExp2val(e2)
		if e2.k == VRELOCABLE && fs.getCode(e2).GetOpCode() == OP_CONCAT {
			LuaAssert(e1.s.info == fs.getCode(e2).GetArgB()-1)
			fs.freeExp(e1)
			fs.getCode(e2).SetArgB(e1.s.info)
			e1.k = VRELOCABLE
			e1.s.info = e2.s.info
		} else {
			fs.kExp2NextReg(e2) /* operand must be on the 'stack' */
			fs.codeArith(OP_CONCAT, e1, e2)
		}
	case OPR_ADD:
		fs.codeArith(OP_ADD, e1, e2)
	case OPR_SUB:
		fs.codeArith(OP_SUB, e1, e2)
	case OPR_MUL:
		fs.codeArith(OP_MUL, e1, e2)
	case OPR_DIV:
		fs.codeArith(OP_DIV, e1, e2)
	case OPR_MOD:
		fs.codeArith(OP_MOD, e1, e2)
	case OPR_POW:
		fs.codeArith(OP_POW, e1, e2)
	case OPR_EQ:
		fs.codeComp(OP_EQ, 1, e1, e2)
	case OPR_NE:
		fs.codeComp(OP_EQ, 0, e1, e2)
	case OPR_LT:
		fs.codeComp(OP_LT, 1, e1, e2)
	case OPR_LE:
		fs.codeComp(OP_LE, 1, e1, e2)
	case OPR_GT:
		fs.codeComp(OP_LT, 0, e1, e2)
	case OPR_GE:
		fs.codeComp(OP_LE, 0, e1, e2)
	default:
		LuaAssert(false)
	}
}

// 对应C函数：`static void codecomp (FuncState *fs, OpCode op, int cond, expdesc *e1, expdesc *e2)'
func (fs *FuncState) codeComp(op OpCode, cond int, e1 *expdesc, e2 *expdesc) {
	var o1 = fs.kExp2RK(e1)
	var o2 = fs.kExp2RK(e2)
	fs.freeExp(e2)
	fs.freeExp(e1)
	if cond == 0 && op != OP_EQ {
		var temp int /* exchange args to replace by `<' or `<=' */
		temp = o1
		o1 = o2
		o2 = temp /* o1 <==> o2 */
		cond = 1
	}
	e1.s.info = fs.condJump(op, cond, o1, o2)
	e1.k = VJMP
}

// 对应C函数：`void luaK_patchlist (FuncState *fs, int list, int target)'
func (fs *FuncState) kPatchList(list int, target int) {
	if target == fs.pc {
		fs.kPatchToHere(list)
	} else {
		LuaAssert(target < fs.pc)
		fs.patchListAux(list, target, NO_REG, target)
	}
}

// 对应C函数：`void luaK_storevar (FuncState *fs, expdesc *var, expdesc *ex)'
func (fs *FuncState) kStoreVar(v *expdesc, ex *expdesc) {
	switch v.k {
	case VLOCAL:
		fs.freeExp(ex)
		fs.exp2reg(ex, v.s.info)
		return
	case VUPVAL:
		var e = fs.kExp2anyReg(ex)
		fs.kCodeABC(OP_SETUPVAL, e, v.s.info, 0)
	case VGLOBAL:
		var e = fs.kExp2anyReg(ex)
		fs.kCodeABx(OP_SETGLOBAL, e, v.s.info)
	case VINDEXED:
		var e = fs.kExp2RK(ex)
		fs.kCodeABC(OP_SETTABLE, v.s.info, v.s.aux, e)
	default:
		LuaAssert(false) /* invalid var kind to store */
	}
	fs.freeExp(ex)
}
