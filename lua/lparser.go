package golua

// FuncState
// state needed to generate code for a given function
// 对应C结构：`struct FuncState'
type FuncState struct {
	f          *Proto                      /* current function header */
	h          *Table                      /* table to find (and reuse) elements in `k' */
	prev       *FuncState                  /* enclosing function */
	ls         *LexState                   /* lexical state */
	L          *LuaState                   /* copy of the Lua state */
	bl         *BlockCnt                   /* chain of current blocks */
	pc         int                         /* next position to code (equivalent to `ncode') */
	lastTarget int                         /* `pc' of last `jump target' */
	jpc        int                         /* list of pending jumps to `pc' */
	freeReg    int                         /* first free register */
	nk         int                         /* number of elements in `k' */
	np         int                         /* number of elements in `p' */
	nLocVars   int                         /* number of elements in `locvars' */
	nActVar    int                         /* number of active local variables */
	upvalues   [LUAI_MAXUPVALUES]upvaldesc /* upvalues */
	actvar     [LUAI_MAXVARS]uint16        /* declared-variable stack */
}

// BlockCnt
// nodes for block list (list of active blocks)
// 对应C结构：`struct BlockCnt'
type BlockCnt struct {
	previous    *BlockCnt /* chain */
	breakList   int       /* list of jumps out of this loop */
	nActVar     int       /* # active locals outside the breakable structure */
	upval       bool      /* true if some variable in the block is an up-value */
	isBreakable bool      /* true if `block' is a loop */
}

// 对应C结构：`struct upvaldesc'
type upvaldesc struct {
	k    expkind
	info int
}

// 对应C结构：`struct expdesc'
type expdesc struct {
	k    expkind
	s    struct{ info, aux int }
	nval LuaNumber
	t    int /* patch list of `exit when true' */
	f    int /* patch list of `exit when false' */
}

type expkind int

const (
	VVOID expkind = iota /* no value */
	VNIL
	VTRUE
	VFALSE
	VK         /* info = index of constant in `k' */
	VKNUM      /* nval = numerical value */
	VLOCAL     /* info = local register */
	VUPVAL     /* info = index of upvalue in `upvalues' */
	VGLOBAL    /* info = index of table; aux = index of global name in `k' */
	VINDEXED   /* info = table register; aux = index register (or `k') */
	VJMP       /* info = instruction pc */
	VRELOCABLE /* info = instruction pc */
	VNONRELOC  /* info = result register */
	VCALL      /* info = instruction pc */
	VVARGARG   /* info = instruction pc */
)

// YParser
// 对应C函数：`Proto *luaY_parser (lua_State *L, ZIO *z, Mbuffer *buff, const char *name)'
func (L *LuaState) YParser(z *ZIO, buff *MBuffer, name []byte) *Proto {
	var lexState LexState
	var funcState FuncState
	lexState.buff = buff
	xSetInput(L, &lexState, z, L.sNew(name))
	funcState.openFunc(&lexState)
	funcState.f.isVarArg = VARARG_ISVARARG /* main func. is always vararg */
	lexState.xNext()                       /* read first token */
	lexState.chunk()
	lexState.check(TK_EOS)
	lexState.closeFunc()
	LuaAssert(funcState.prev == nil)
	LuaAssert(funcState.f.nUps == 0)
	LuaAssert(lexState.fs == nil)
	return funcState.f
}

// 对应C函数：`static void openFunc (LexState *ls, FuncState *fs)'
func (fs *FuncState) openFunc(ls *LexState) {
	var L = ls.L
	var f = L.fNewProto()
	fs.f = f
	fs.prev = ls.fs /* linked list of funcstates */
	fs.ls = ls
	fs.L = L
	ls.fs = fs
	fs.pc = 0
	fs.lastTarget = -1
	fs.jpc = NO_JUMP
	fs.freeReg = 0
	fs.nk = 0
	fs.np = 0
	fs.nLocVars = 0
	fs.nActVar = 0
	fs.bl = nil
	f.source = ls.source
	f.maxStackSize = 2 /* registers 0/1 are always valid */
	fs.h = L.hNew(0, 0)
	/* anchor table of constants and prototype (to avoid being collected) */
	L.Top().SetTable(L, fs.h)
	L.IncTop()
	L.Top().SetAny(f)
	L.IncTop()
}

// 对应C函数：`static void close_func (LexState *ls)'
func (ls *LexState) closeFunc() {
	var L = ls.L
	var fs = ls.fs
	var f = fs.f
	ls.removeVars(0)
	fs.kRet(0, 0) /* final return */
	mReallocVector(L, &f.code, f.sizeCode, fs.pc)
	f.sizeCode = fs.pc
	mReallocVector(L, &f.lineInfo, f.sizeLineInfo, fs.pc)
	f.sizeLineInfo = fs.pc
	mReallocVector(L, &f.k, f.sizeK, fs.nk)
	f.sizeK = fs.nk
	mReallocVector(L, &f.p, f.sizeP, fs.np)
	f.sizeP = fs.np
	mReallocVector(L, &f.locVars, f.sizeLocVars, fs.nLocVars)
	f.sizeLocVars = fs.nLocVars
	mReallocVector(L, &f.upValues, f.sizeUpValues, f.nUps)
	f.sizeUpValues = f.nUps
	LuaAssert(f.gCheckCode())
	LuaAssert(fs.bl == nil)
	ls.fs = fs.prev
	L.top -= 2 /* remove table and prototype from the stack */
	/* last token read was anchored in defunct function; must re-anchor it */
	if fs != nil {
		ls.anchorToken()
	}
}

// 对应C函数：`static void pushclosure (LexState *ls, FuncState *func, expdesc *v)'
func (ls *LexState) pushClosure(fn *FuncState, v *expdesc) {
	var fs = ls.fs
	var f = fs.f
	var oldSize = f.sizeP
	mGrowVector(ls.L, &f.p, fs.np, &f.sizeP, MAXARG_Bx, "constant table overflow")
	for i := oldSize; i < f.sizeP; i++ {
		f.p[i] = nil
	}
	f.p[fs.np] = fn.f
	fs.np++
	ls.L.cObjBarrier(f, fn.f)
	v.initExp(VRELOCABLE, fs.kCodeABx(OP_CLOSURE, 0, fs.np-1))
	for i := 0; i < fn.f.nUps; i++ {
		var o OpCode
		if fn.upvalues[i].k == VLOCAL {
			o = OP_MOVE
		} else {
			o = OP_GETUPVAL
		}
		fs.kCodeABC(o, 0, int(fn.upvalues[i].info), 0)
	}
}

// 对应C函数：`static void chunk (LexState *ls)'
func (ls *LexState) chunk() {
	/* chunk -> { stat [`;'] } */
	var isLast = false
	ls.enterLevel()
	for !isLast && !blockFollow(ls.t.token) {
		isLast = ls.statement()
		ls.testNext(';')
		LuaAssert(ls.fs.f.maxStackSize >= ls.fs.freeReg &&
			ls.fs.freeReg >= ls.fs.nActVar)
		ls.fs.freeReg = ls.fs.nActVar /* free registers */
	}
	ls.leaveLevel()
}

// 对应C函数：`static void enterlevel (LexState *ls)'
func (ls *LexState) enterLevel() {
	ls.L.nCCalls++
	if ls.L.nCCalls > LUAI_MAXCCALLS {
		ls.xLexError("chunk has too many syntax levels", 0)
	}
}

// 对应C函数：`leavelevel(ls)'
func (ls *LexState) leaveLevel() {
	ls.L.nCCalls--
}

// 对应C函数：`static int block_follow (int token)'
func blockFollow(token tk) bool {
	switch token {
	case TK_ELSE, TK_ELSEIF, TK_END, TK_UNTIL, TK_EOS:
		return true
	default:
		return false
	}
}

// 对应C函数：`static int statement (LexState *ls)'
func (ls *LexState) statement() (lastStat bool) {
	var line = ls.lineNumber /* may be needed for error messages */
	switch ls.t.token {
	case TK_IF: /* stat -> ifstat */
		ls.ifStat(line)
		return false
	case TK_WHILE: /* stat -> whilestat */
		ls.whileStat(line)
		return false
	case TK_DO: /* stat -> DO block END */
		ls.xNext() /* skip DO */
		ls.block()
		ls.checkMatch(TK_END, TK_DO, line)
		return false
	case TK_FOR: /* stat -> forstat */
		ls.forStat(line)
		return false
	case TK_REPEAT: /* stat -> repreatstat */
		ls.repeatStat(line)
		return false
	case TK_FUNCTION:
		ls.funcStat(line) /* stat -> funcstat */
		return false
	case TK_LOCAL: /* stat -> localstat */
		ls.xNext()                    /* skip LOCAL */
		if ls.testNext(TK_FUNCTION) { /* local function? */
			ls.localFunc()
		} else {
			ls.localStat()
		}
		return false
	case TK_RETURN: /* stat -> retstat */
		ls.retStat()
		return true /* must be last statement */
	case TK_BREAK: /* stat -> breakstat */
		ls.xNext() /* skip BREAK */
		ls.breakStat()
		return true /* must be last statement */
	default:
		ls.exprStat()
		return false /* to avoid warnings */
	}
}

// 对应C函数：`static void ifstat (LexState *ls, int line)'
func (ls *LexState) ifStat(line int) {
	/* ifstat -> IF cond THEN block {ELSEIF cond THEN block} [ELSE block] END */
	var fs = ls.fs
	var escapseList = NO_JUMP
	var flist = ls.testThenBlock() /* IF cond THEN block */
	for ls.t.token == TK_ELSEIF {
		fs.kConcat(&escapseList, fs.kJump())
		fs.kPatchToHere(flist)
		flist = ls.testThenBlock() /* ELSEIF cond THEN block */
	}
	if ls.t.token == TK_ELSE {
		fs.kConcat(&escapseList, fs.kJump())
		fs.kPatchToHere(flist)
		ls.xNext() /* skip ELSE (after patch, for correct line info) */
		ls.block() /* `else' part */
	} else {
		fs.kConcat(&escapseList, flist)
	}
	fs.kPatchToHere(escapseList)
	ls.checkMatch(TK_END, TK_IF, line)
}

// 对应C函数：`static int test_then_block (LexState *ls)'
func (ls *LexState) testThenBlock() int {
	/* test_then_block -> [IF | ELSEIF] cond THEN block */
	ls.xNext() /* skip IF or ELSEIF */
	var condExit = ls.cond()
	ls.checkNextX(TK_THEN)
	ls.block() /* `then' part */
	return condExit
}

// 对应C函数：`static int cond (LexState *ls)'
func (ls *LexState) cond() int {
	/* cond -> exp */
	var v = &expdesc{}
	ls.expr(v) /* read condition */
	if v.k == VNIL {
		v.k = VFALSE /* `false' are all equal here */
	}
	ls.fs.kGoIfTrue(v)
	return v.f
}

// 对应C函数：`static void expr (LexState *ls, expdesc *v)'
func (ls *LexState) expr(v *expdesc) {
	ls.subExpr(v, 0)
}

const UNARY_PRIORITY = 8 /* priority for unary operators */

// subExpr -> (simpleexp | unop subExpr) { binop subExpr }
// where `binop' is any binary operator with a priority higher than `limit'
// 对应C函数：`static BinOpr subExpr (LexState *ls, expdesc *v, unsigned int limit)'
func (ls *LexState) subExpr(v *expdesc, limit int) BinOpr {
	ls.enterLevel()
	if uop := getunopr(ls.t.token); uop != OPR_NOUNOPR {
		ls.xNext()
		ls.subExpr(v, UNARY_PRIORITY)
		ls.fs.kPrefix(uop, v)
	} else {
		ls.simpleExp(v)
	}
	/* expand while operators have priorities higher than `limit' */
	var op = getbinopr(ls.t.token)
	for op != OPR_NOBINOPR && priority[op].left > int(limit) {
		ls.xNext()
		ls.fs.kInfix(op, v)
		/* read sub-expression with higher priority */
		var v2 = &expdesc{}
		var nextOp = ls.subExpr(v2, priority[op].right)
		ls.fs.kPosFix(op, v, v2)
		op = nextOp
	}
	ls.leaveLevel()
	return op /* return first untreated operator */
}

// 对应C函数：`static UnOpr getunopr (int op)'
func getunopr(op tk) UnOpr {
	switch op {
	case TK_NOT:
		return OPR_NOT
	case '-':
		return OPR_MINUS
	case '#':
		return OPR_LEN
	default:
		return OPR_NOUNOPR
	}
}

// 对应C函数：`static BinOpr getbinopr (int op)'
func getbinopr(op tk) BinOpr {
	switch op {
	case '+':
		return OPR_ADD
	case '-':
		return OPR_SUB
	case '*':
		return OPR_MUL
	case '/':
		return OPR_DIV
	case '%':
		return OPR_MOD
	case '^':
		return OPR_POW
	case TK_CONCAT:
		return OPR_CONCAT
	case TK_NE:
		return OPR_NE
	case TK_EQ:
		return OPR_EQ
	case '<':
		return OPR_LT
	case TK_LE:
		return OPR_LE
	case '>':
		return OPR_GT
	case TK_GE:
		return OPR_GE
	case TK_AND:
		return OPR_AND
	case TK_OR:
		return OPR_OR
	default:
		return OPR_NOBINOPR
	}
}

var priority = []struct {
	left  int /* left priority for each binary operator */
	right int /* right priority */
}{
	{6, 6}, {6, 6}, {7, 7}, {7, 7}, {7, 7}, /* `+' `-' `*' `/' `%'  */
	{10, 9}, {5, 4}, /* power and concat (right associative) */
	{3, 3}, {3, 3}, /* equality and inequality */
	{3, 3}, {3, 3}, {3, 3}, {3, 3}, /* order */
	{2, 2}, {1, 1}, /* logical (and/or) */
} /* ORDER OPR */

// 对应C函数：`static void simpleexp (LexState *ls, expdesc *v)'
func (ls *LexState) simpleExp(v *expdesc) {
	/* simpleexp -> NUMBER | STRING | NIL | true | false | ... |
	   constructor | FUNCTION body | primaryexp */
	switch ls.t.token {
	case TK_NUMBER:
		v.initExp(VKNUM, 0)
		v.nval = ls.t.semInfo.r
	case TK_STRING:
		ls.codeString(v, ls.t.semInfo.ts)
	case TK_NIL:
		v.initExp(VNIL, 0)
	case TK_TRUE:
		v.initExp(VTRUE, 0)
	case TK_FALSE:
		v.initExp(VFALSE, 0)
	case TK_DOTS: /* vararg */
		var fs = ls.fs
		ls.checkCondition(fs.f.isVarArg != 0,
			"cannot use '...' outside a vararg function")
		fs.f.isVarArg &= ^lu_byte(VARARG_NEEDSARG) /* don't need 'arg' */
		v.initExp(VVARGARG, fs.kCodeABC(OP_VARARG, 0, 1, 0))
	case '{': /* constructor */
		ls.constructor(v)
		return
	case TK_FUNCTION:
		ls.xNext()
		ls.body(v, false, ls.lineNumber)
		return
	default:
		ls.primaryExp(v)
		return
	}
	ls.xNext()
}

// 对应C函数：`static void codestring (LexState *ls, expdesc *e, TString *s)'
func (ls *LexState) codeString(e *expdesc, s *TString) {
	e.initExp(VK, ls.fs.kStringK(s))
}

// 对应C函数：`check_condition(ls,c,msg)'
func (ls *LexState) checkCondition(c bool, msg string) {
	if !c {
		ls.xSyntaxError(msg)
	}
}

// 对应C函数：`static void constructor (LexState *ls, expdesc *t)'
func (ls *LexState) constructor(t *expdesc) {
	/* constructor -> ?? */
	var fs = ls.fs
	var line = ls.lineNumber
	var pc = fs.kCodeABC(OP_NEWTABLE, 0, 0, 0)
	var cc = &ConsControl{
		t:       t,
		nh:      0,
		na:      0,
		tostore: 0,
	}
	t.initExp(VRELOCABLE, pc)
	cc.v.initExp(VVOID, 0) /* no value (yet) */
	ls.fs.kExp2NextReg(t)  /* fix it at stack top (for gc) */
	ls.checkNextX('{')
	for {
		LuaAssert(cc.v.k == VVOID || cc.tostore > 0)
		if ls.t.token == '}' {
			break
		}
		fs.closeListField(cc)
		switch ls.t.token {
		case TK_NAME: /* amy be list-fields or recfields */
			ls.xLookAhead()
			if ls.lookAhead.token != '=' { /* expression? */
				ls.listField(cc)
			} else {
				ls.recField(cc)
			}
		case '[': /* constructor_item -> recfield */
			ls.recField(cc)
		default: /* constructor_part -> listfield */
			ls.listField(cc)
		}

		if !ls.testNext(',') && !ls.testNext(';') {
			break
		}
	}
	ls.checkMatch('}', '{', line)
	fs.lastListField(cc)
	fs.f.code[pc].SetArgB(oInt2Fb(uint(cc.na))) /* set initial array size */
	fs.f.code[pc].SetArgC(oInt2Fb(uint(cc.nh))) /* set initial table size */
}

type ConsControl struct {
	v       expdesc  /* last list item read */
	t       *expdesc /* table descriptor */
	nh      int      /* total number of `record' elements */
	na      int      /* total number of array elements */
	tostore int      /* number of array elements pending to be stored */
}

// 对应C函数：`static void recfield (LexState *ls, struct ConsControl *cc)'
func (ls *LexState) recField(cc *ConsControl) {
	/* recfield -> (NAME | [exp1]) = exp1 */
	var fs = ls.fs
	var reg = ls.fs.freeReg
	var key, val expdesc
	// var rkkey int
	if ls.t.token == TK_NAME {
		fs.yCheckLimit(cc.nh, MAX_INT, "items in a constructor")
		ls.checkName(&key)
	} else { /* ls->t.token == '[' */
		ls.yIndex(&key)
	}
	cc.nh++
	ls.checkNextX('=')
	var rkkey = fs.kExp2RK(&key)
	ls.expr(&val)
	fs.kCodeABC(OP_SETTABLE, cc.t.s.info, rkkey, fs.kExp2RK(&val))
	fs.freeReg = reg /* free registers */
}

// 对应C函数：`static void closelistfield (FuncState *fs, struct ConsControl *cc)'
func (fs *FuncState) closeListField(cc *ConsControl) {
	if cc.v.k == VVOID { /* there is no list item */
		return
	}
	fs.kExp2NextReg(&cc.v)
	cc.v.k = VVOID
	if cc.tostore == LFIELDS_PER_FLUSH {
		fs.kSetList(cc.t.s.info, cc.na, cc.tostore) /* flush */
		cc.tostore = 0                              /* no more items pending */
	}
}

// 对应C函数：`static void listfield (LexState *ls, struct ConsControl *cc)'
func (ls *LexState) listField(cc *ConsControl) {
	ls.expr(&cc.v)
	ls.fs.yCheckLimit(cc.na, MAX_INT, "items in a constructor")
	cc.na++
	cc.tostore++
}

// 对应C函数：`static void lastlistfield (FuncState *fs, struct ConsControl *cc)'
func (fs *FuncState) lastListField(cc *ConsControl) {
	if cc.tostore == 0 {
		return
	}
	if hasMultiRet(cc.v.k) {
		fs.kSetMultRet(&cc.v)
		fs.kSetList(cc.t.s.info, cc.na, LUA_MULTRET)
		cc.na-- /* do not count last expression (unknown number of elements */
	} else {
		if cc.v.k != VVOID {
			fs.kExp2NextReg(&cc.v)
		}
		fs.kSetList(cc.t.s.info, cc.na, cc.tostore)
	}
}

// 对应C函数：`static void checknext (LexState *ls, int c)'
func (ls *LexState) checkNextX(c tk) {
	ls.check(c)
	ls.xNext()
}

// 对应C函数：`static void check (LexState *ls, int c)'
func (ls *LexState) check(c tk) {
	if ls.t.token != c {
		ls.errorExpected(c)
	}
}

// 对应C函数：`static void error_expected (LexState *ls, int token)'
func (ls *LexState) errorExpected(token tk) {
	ls.xSyntaxError(
		string(ls.L.oPushFString(LUA_QS+" expected", ls.xToken2str(token))))
}

// 对应C函数：`static void errorlimit (FuncState *fs, int limit, const char *what)'
func (fs *FuncState) errorLimit(limit int, what string) {
	var msg []byte
	if fs.f.lineDefined == 0 {
		msg = fs.L.oPushFString("main function has more than %d %s", limit, what)
	} else {
		msg = fs.L.oPushFString("function at line %d has more than %d %s",
			fs.f.lineDefined, limit, what)
	}
	fs.ls.xLexError(string(msg), 0)
}

// 对应C函数：`static int testnext (LexState *ls, int c)'
func (ls *LexState) testNext(c tk) bool {
	if ls.t.token == c {
		ls.xNext()
		return true
	}
	return false
}

// 对应C函数：` luaY_checklimit(fs,v,l,m)'
func (fs *FuncState) yCheckLimit(v int, limit int, what string) {
	if v > limit {
		fs.errorLimit(limit, what)
	}
}

// 对应C函数：`static void checkname(LexState *ls, expdesc *e)'
func (ls *LexState) checkName(e *expdesc) {
	ls.codeString(e, ls.strCheckName())
}

// 对应C函数：`static TString *str_checkname (LexState *ls)'
func (ls *LexState) strCheckName() *TString {
	ls.check(TK_NAME)
	var ts = ls.t.semInfo.ts
	ls.xNext()
	return ts
}

// 对应C函数：`static void yindex (LexState *ls, expdesc *v)'
func (ls *LexState) yIndex(v *expdesc) {
	/* index -> [ expr ] */
	ls.xNext() /* skip the '[' */
	ls.expr(v)
	ls.fs.kExp2val(v)
	ls.checkNextX(']')
}

// 对应C函数：`static void check_match (LexState *ls, int what, int who, int where)'
func (ls *LexState) checkMatch(what tk, who tk, where int) {
	if !ls.testNext(what) {
		if where == ls.lineNumber {
			ls.errorExpected(what)
		} else {
			var msg = ls.L.oPushFString("'%s' expected (to close '%s' at line %d)",
				ls.xToken2str(what), ls.xToken2str(who), where)
			ls.xSyntaxError(string(msg))
		}
	}
}

// 对应C函数：`hasmultret(k)'
func hasMultiRet(k expkind) bool {
	return k == VCALL || k == VVARGARG
}

// 对应C函数：`static void body (LexState *ls, expdesc *e, int needself, int line)'
func (ls *LexState) body(e *expdesc, needSelf bool, line int) {
	/*body -> `(' parlist `)' chunk END */
	var newFS = &FuncState{}
	newFS.openFunc(ls)
	newFS.f.lineDefined = line
	ls.checkNextX('(')
	if needSelf {
		ls.newLocalVarLiteral("self", 0)
		ls.adjustLocalVars(1)
	}
	ls.parList()
	ls.checkNextX(')')
	ls.chunk()
	newFS.f.lastLineDefined = ls.lineNumber
	ls.checkMatch(TK_END, TK_FUNCTION, line)
	ls.closeFunc()
	ls.pushClosure(newFS, e)
}

// 对应C函数：`new_localvarliteral(ls,v,n)'
func (ls *LexState) newLocalVarLiteral(v string, n int) {
	ls.newLocalVar(ls.xNewString([]byte(v)), n)
}

// 对应C函数：`static void new_localvar (LexState *ls, TString *name, int n)'
func (ls *LexState) newLocalVar(name *TString, n int) {
	var fs = ls.fs
	fs.yCheckLimit(fs.nActVar+n+1, LUAI_MAXVARS, "local variables")
	fs.actvar[fs.nActVar+n] = uint16(ls.registerLocalVar(name))
}

// 对应C函数：`static int registerlocalvar (LexState *ls, TString *varname)'
func (ls *LexState) registerLocalVar(varName *TString) int {
	var fs = ls.fs
	var f = fs.f
	var oldSize = f.sizeLocVars
	mGrowVector(ls.L, &f.locVars, fs.nLocVars, &f.sizeLocVars,
		SHRT_MAX, "too many local variables")
	for oldSize < f.sizeLocVars {
		f.locVars[oldSize].varName = nil
		oldSize++
	}
	f.locVars[fs.nLocVars].varName = varName
	ls.L.cObjBarrier(f, varName)
	fs.nLocVars++
	return fs.nLocVars - 1
}

// 对应C函数：`static void adjustlocalvars (LexState *ls, int nvars)'
func (ls *LexState) adjustLocalVars(nVars int) {
	var fs = ls.fs
	fs.nActVar = fs.nActVar + nVars
	for ; nVars > 0; nVars-- {
		fs.getLocVar(fs.nActVar - nVars).startPc = fs.pc
	}
}

// 对应C函数：`getlocvar(fs, i)'
func (fs *FuncState) getLocVar(i int) *LocVar {
	return &fs.f.locVars[fs.actvar[i]]
}

// 对应C函数：`static void parlist (LexState *ls)'
func (ls *LexState) parList() {
	/* parlist -> [ param { `,' param } ] */
	var fs = ls.fs
	var f = fs.f
	var nParams = 0
	f.isVarArg = 0
	if ls.t.token != ')' { /* is `parlist' not empty? */
		for {
			switch ls.t.token {
			case TK_NAME: /* param -> NAME */
				ls.newLocalVar(ls.strCheckName(), nParams)
				nParams++
			case TK_DOTS: /* param -> `...' */
				ls.xNext()
				if LUA_COMPAT_VARARG {
					/* use `arg' as default name */
					ls.newLocalVarLiteral("arg", nParams)
					f.isVarArg = VARARG_HASARG | VARARG_NEEDSARG
				}
				f.isVarArg |= VARARG_ISVARARG
			default:
				ls.xSyntaxError("<name> or '...' expected")

			}

			if f.isVarArg != 0 || !ls.testNext(',') {
				break
			}
		}
	}
	ls.adjustLocalVars(nParams)
	f.numParams = fs.nActVar - int(f.isVarArg&VARARG_HASARG)
	fs.kReserveRegs(fs.nActVar) /* reserve register for parameters */
}

// 对应C函数：`static void removevars (LexState *ls, int tolevel)'
func (ls *LexState) removeVars(toLevel int) {
	var fs = ls.fs
	for fs.nActVar > toLevel {
		fs.nActVar--
		fs.getLocVar(fs.nActVar).endPc = fs.pc
	}
}

// 对应C函数：`static void anchor_token (LexState *ls)'
func (ls *LexState) anchorToken() {
	if ls.t.token == TK_NAME || ls.t.token == TK_STRING {
		var ts = ls.t.semInfo.ts
		ls.xNewString(ts.GetStr())
	}
}

// 对应C函数：·static void primaryexp (LexState *ls, expdesc *v)‘
func (ls *LexState) primaryExp(v *expdesc) {
	/* primaryexp ->
	 *      prefixexp { `.' NAME | `[' exp `]' | `:' NAME funcargs | funcargs } */
	var fs = ls.fs
	ls.prefixExp(v)
	for {
		switch ls.t.token {
		case '.': /* field */
			ls.field(v)
		case '[': /* `[' exp1 `]' */
			var key = &expdesc{}
			fs.kExp2anyReg(v)
			ls.yIndex(key)
			fs.kIndexed(v, key)
		case ':': /* `:' NAME funcargs */
			var key = &expdesc{}
			ls.xNext()
			ls.checkName(key)
			fs.kSelf(v, key)
			ls.funcArgs(v)
		case '(', TK_STRING, '{': /* funcargs */
			fs.kExp2NextReg(v)
			ls.funcArgs(v)
		default:
			return
		}
	}
}

// 对应C函数：`static void prefixexp (LexState *ls, expdesc *v)'
func (ls *LexState) prefixExp(v *expdesc) {
	/* prefixexp -> NAME | '(' expr ')' */
	switch ls.t.token {
	case '(':
		var line = ls.lineNumber
		ls.xNext()
		ls.expr(v)
		ls.checkMatch(')', '(', line)
		ls.fs.kDischargeVars(v)
		return
	case TK_NAME:
		ls.singleVar(v)
		return
	default:
		ls.xSyntaxError("unexpected symbol")
		return
	}
}

// 对应C函数：`static void singlevar (LexState *ls, expdesc *var)'
func (ls *LexState) singleVar(v *expdesc) {
	var varName = ls.strCheckName()
	var fs = ls.fs
	if singleVarAux(fs, varName, v, 1) == VGLOBAL {
		v.s.info = fs.kStringK(varName) /* info points to global name */
	}
}

// 对应C函数：`static int singlevaraux (FuncState *fs, TString *n, expdesc *var, int base)'
func singleVarAux(fs *FuncState, n *TString, va *expdesc, base int) expkind {
	if fs == nil { /* no more levels? */
		va.initExp(VGLOBAL, NO_REG) /* default is global variable */
		return VGLOBAL
	}
	var v = fs.searchVar(n) /* look up at current level */
	if v >= 0 {
		va.initExp(VLOCAL, v)
		if base != 0 {
			fs.markUpval(v) /* local will be used as an upval */
		}
		return VLOCAL
	} else { /* not found at current level; try upper one */
		if singleVarAux(fs.prev, n, va, 0) == VGLOBAL {
			return VGLOBAL
		}
		va.s.info = fs.indexUpValue(n, va) /* else was LOCAL or UPVAL */
		va.k = VUPVAL                      /* upvalue in this level */
		return VUPVAL
	}
}

// 对应C函数：`static int searchvar (FuncState *fs, TString *n)'
func (fs *FuncState) searchVar(n *TString) int {
	for i := fs.nActVar - 1; i >= 0; i-- {
		if n == fs.getLocVar(i).varName {
			return i
		}
	}
	return -1 /* not found */
}

// 对应C函数：`static void markupval (FuncState *fs, int level)'
func (fs *FuncState) markUpval(level int) {
	var bl = fs.bl
	for bl != nil && bl.nActVar > level {
		bl = bl.previous
	}
	if bl != nil {
		bl.upval = true
	}
}

// 对应C函数：`static int indexupvalue (FuncState *fs, TString *name, expdesc *v)'
func (fs *FuncState) indexUpValue(name *TString, v *expdesc) int {
	var f = fs.f
	var oldSize = f.sizeUpValues
	for i := 0; i < f.nUps; i++ {
		if fs.upvalues[i].k == v.k && fs.upvalues[i].info == v.s.info {
			LuaAssert(f.upValues[i] == name)
			return i
		}
	}
	/* new one */
	fs.yCheckLimit(f.nUps+1, LUAI_MAXUPVALUES, "upvalues")
	mGrowVector(fs.L, &f.upValues, f.nUps, &f.sizeUpValues, MAX_INT, "")
	for i := oldSize; i < f.sizeUpValues; i++ {
		f.upValues[i] = nil
	}
	f.upValues[f.nUps] = name
	fs.L.cObjBarrier(f, name)
	LuaAssert(v.k == VLOCAL || v.k == VUPVAL)
	fs.upvalues[f.nUps].k = v.k
	fs.upvalues[f.nUps].info = v.s.info
	f.nUps++
	return f.nUps - 1
}

// 对应C函数：`static void field (LexState *ls, expdesc *v)'
func (ls *LexState) field(v *expdesc) {
	/* field -> ['.' | ':'] NAME */
	var fs = ls.fs
	var key = &expdesc{}
	fs.kExp2anyReg(v)
	ls.xNext() /* skip the dot or colon */
	ls.checkName(key)
	fs.kIndexed(v, key)
}

// 对应C函数：`void luaK_indexed (FuncState *fs, expdesc *t, expdesc *k)'
func (fs *FuncState) kIndexed(t *expdesc, k *expdesc) {
	t.s.aux = fs.kExp2RK(k)
	t.k = VINDEXED
}

// 对应C函数：`static void funcargs (LexState *ls, expdesc *f)'
func (ls *LexState) funcArgs(f *expdesc) {
	var fs = ls.fs
	var args = &expdesc{}
	var line = ls.lineNumber
	switch ls.t.token {
	case '(': /* funcargs -> `(' [ explist1 ] `)' */
		if line != ls.lastLine {
			ls.xSyntaxError("ambiguous syntax (function call x new statement)")
		}
		ls.xNext()
		if ls.t.token == ')' { /* arg list is empty? */
			args.k = VVOID
		} else {
			ls.expList1(args)
			fs.kSetMultRet(args)
		}
		ls.checkMatch(')', '(', line)
	case '{': /* funcargs -> constructor */
		ls.constructor(args)
	case TK_STRING: /* funcargs -> STRING */
		ls.codeString(args, ls.t.semInfo.ts)
		ls.xNext() /* must use `seminfo' before `next' */
	default:
		ls.xSyntaxError("function arguments expected")
	}
	LuaAssert(f.k == VNONRELOC)
	var base = f.s.info /* base register for call */
	var nParams int
	if hasMultiRet(args.k) {
		nParams = LUA_MULTRET /* open call */
	} else {
		if args.k != VVOID {
			fs.kExp2NextReg(args) /* close last argument */
		}
		nParams = fs.freeReg - (base + 1)
	}
	f.initExp(VCALL, fs.kCodeABC(OP_CALL, base, nParams+1, 2))
	fs.kFixLine(line)
	fs.freeReg = base + 1 /* call remove function and arguments and leaves (unless changed) one result */
}

// 对应C函数：`static int explist1 (LexState *ls, expdesc *v)'
func (ls *LexState) expList1(v *expdesc) int {
	/* explist1 -> expr { `,' expr } */
	var n = 1 /* at least one expression */
	ls.expr(v)
	for ls.testNext(',') {
		ls.fs.kExp2NextReg(v)
		ls.expr(v)
		n++
	}
	return n
}

// 对应C函数：`static void block (LexState *ls)'
func (ls *LexState) block() {
	/* block -> chunk */
	var fs = ls.fs
	var bl = &BlockCnt{}
	fs.enterBlock(bl, false)
	ls.chunk()
	LuaAssert(bl.breakList == NO_JUMP)
	fs.leaveBlock()
}

// 对应C函数：`static void enterblock (FuncState *fs, BlockCnt *bl, lu_byte isbreakable)'
func (fs *FuncState) enterBlock(bl *BlockCnt, isBreakable bool) {
	bl.breakList = NO_JUMP
	bl.isBreakable = isBreakable
	bl.nActVar = fs.nActVar
	bl.upval = false
	bl.previous = fs.bl
	fs.bl = bl
	LuaAssert(fs.freeReg == fs.nActVar)
}

// 对应C函数：`static void leaveblock (FuncState *fs)'
func (fs *FuncState) leaveBlock() {
	var bl = fs.bl
	fs.bl = bl.previous
	fs.ls.removeVars(bl.nActVar)
	if bl.upval {
		fs.kCodeABC(OP_CLOSE, bl.nActVar, 0, 0)
	}
	/* a block either controls scope or breaks (never both) */
	LuaAssert(!bl.isBreakable || !bl.upval)
	LuaAssert(bl.nActVar == fs.nActVar)
	fs.freeReg = fs.nActVar /* free registers */
	fs.kPatchToHere(bl.breakList)
}

// 对应C函数：`static void whilestat (LexState *ls, int line)'
func (ls *LexState) whileStat(line int) {
	/* whilestat -> WHILE cond DO block END */
	var fs = ls.fs
	var bl = &BlockCnt{}
	ls.xNext() /* skip WHILE */
	var whileInit = fs.kGetLabel()
	var condExit = ls.cond()
	fs.enterBlock(bl, true)
	ls.checkNextX(TK_DO)
	ls.block()
	fs.kPatchList(fs.kJump(), whileInit)
	ls.checkMatch(TK_END, TK_WHILE, line)
	fs.leaveBlock()
	fs.kPatchToHere(condExit) /* false conditions finish the loop */
}

// 对应C函数：`static void forstat (LexState *ls, int line)'
func (ls *LexState) forStat(line int) {
	/* forstat -> FOR (fornum | forlist) END*/
	var fs = ls.fs
	var bl = &BlockCnt{}
	fs.enterBlock(bl, true)         /* scope for loop and control variables */
	ls.xNext()                      /* skip `for' */
	var varName = ls.strCheckName() /* first variable name */
	switch ls.t.token {
	case '=':
		ls.forNum(varName, line)
	case ',', TK_IN:
		ls.forList(varName)
	default:
		ls.xSyntaxError("'=' or 'in' expected")
	}
	ls.checkMatch(TK_END, TK_FOR, line)
	fs.leaveBlock() /* loop scope (`break' jumps to this point) */
}

// 对应C函数：`static void fornum (LexState *ls, TString *varname, int line)'
func (ls *LexState) forNum(varName *TString, line int) {
	/* fornum -> NAME = exp1,exp1[,exp1] forbody */
	var fs = ls.fs
	var base = fs.freeReg
	ls.newLocalVarLiteral("(for index)", 0)
	ls.newLocalVarLiteral("(for limit)", 1)
	ls.newLocalVarLiteral("(for step)", 2)
	ls.newLocalVar(varName, 3)
	ls.checkNextX('=')
	ls.exp1() /* initial value */
	ls.checkNextX(',')
	ls.exp1() /* limit */
	if ls.testNext(',') {
		ls.exp1() /* optional step */
	} else { /* default step = 1 */
		fs.kCodeABx(OP_LOADK, fs.freeReg, fs.kNumberK(1))
		fs.kReserveRegs(1)
	}
	ls.forBody(base, line, 1, true)
}

// 对应C函数：`static int exp1 (LexState *ls)'
func (ls *LexState) exp1() expkind {
	var e = &expdesc{}
	ls.expr(e)
	var k = e.k
	ls.fs.kExp2NextReg(e)
	return k
}

// 对应C函数：`static void forbody (LexState *ls, int base, int line, int nvars, int isnum)'
func (ls *LexState) forBody(base int, line int, nVars int, isNum bool) {
	/* forbody -> DO block */
	var bl = &BlockCnt{}
	var fs = ls.fs
	ls.adjustLocalVars(3) /* control variables */
	ls.checkNextX(TK_DO)
	var prep int
	if isNum {
		prep = fs.kCodeAsBx(OP_FORPREP, base, NO_JUMP)
	} else {
		prep = fs.kJump()
	}
	fs.enterBlock(bl, false) /* scope for declared variables */
	ls.adjustLocalVars(nVars)
	fs.kReserveRegs(nVars)
	ls.block()
	fs.leaveBlock() /* end of scope for declared variables */
	fs.kPatchToHere(prep)
	var endFor int
	if isNum {
		endFor = fs.kCodeAsBx(OP_FORLOOP, base, NO_JUMP)
	} else {
		endFor = fs.kCodeABC(OP_TFORLOOP, base, 0, nVars)
	}
	fs.kFixLine(line) /* pretend that `OP_FOR' starts the loop */
	if isNum {
		fs.kPatchList(endFor, prep+1)
	} else {
		fs.kPatchList(fs.kJump(), prep+1)
	}
}

// 对应C函数：`static void forlist (LexState *ls, TString *indexname)'
func (ls *LexState) forList(indexName *TString) {
	/* forlist -> NAME {,NAME} IN explist1 forbody */
	var fs = ls.fs
	var base = fs.freeReg
	/* create control variables */
	ls.newLocalVarLiteral("(for generator)", 0)
	ls.newLocalVarLiteral("(for state)", 1)
	ls.newLocalVarLiteral("(for control)", 2)
	/* create declared variables */
	ls.newLocalVar(indexName, 3)
	var nVars = 4
	for ls.testNext(',') {
		ls.newLocalVar(ls.strCheckName(), nVars)
		nVars++
	}
	ls.checkNextX(TK_IN)
	var line = ls.lineNumber
	var e = &expdesc{}
	ls.adjustAssign(3, ls.expList1(e), e)
	fs.kCheckStack(3) /* extra space to call generator */
	ls.forBody(base, line, nVars-3, false)
}

// 对应C函数：`static void adjust_assign (LexState *ls, int nvars, int nexps, expdesc *e)'
func (ls *LexState) adjustAssign(nVars int, nExps int, e *expdesc) {
	var fs = ls.fs
	var extra = nVars - nExps
	if hasMultiRet(e.k) {
		extra++ /* includes call itself */
		if extra < 0 {
			extra = 0
		}
		fs.kSetReturns(e, extra) /* last exp. provides the difference */
		if extra > 1 {
			fs.kReserveRegs(extra - 1)
		}
	} else {
		if e.k != VVOID {
			fs.kExp2NextReg(e) /* close last expression */
			if extra > 0 {
				var reg = fs.freeReg
				fs.kReserveRegs(extra)
				fs.kNil(reg, extra)
			}
		}
	}
}

// 对应C函数：`static void repeatstat (LexState *ls, int line)'
func (ls *LexState) repeatStat(line int) {
	/* repeatstat -> REPEAT block UNTIL cond */
	var fs = ls.fs
	var repeatInit = fs.kGetLabel()
	var bl1, bl2 = new(BlockCnt), new(BlockCnt)
	fs.enterBlock(bl1, true)  /* loop block */
	fs.enterBlock(bl2, false) /* scope block */
	ls.xNext()                /* skip REPEAT */
	ls.chunk()
	ls.checkMatch(TK_UNTIL, TK_REPEAT, line)
	var condExit = ls.cond() /* read condition (inside scope block) */

	if !bl2.upval { /* no up-values? */
		fs.leaveBlock()                        /* finish scope */
		ls.fs.kPatchList(condExit, repeatInit) /* close the loop */
	} else { /* complete semantics when there are upvalues */
		ls.breakStat()                           /* if condition then break */
		ls.fs.kPatchToHere(condExit)             /* else... */
		fs.leaveBlock()                          /* finish scope... */
		ls.fs.kPatchList(fs.kJump(), repeatInit) /* and repeat */
	}
	fs.leaveBlock() /* finish loop */
}

// 对应C函数：`static void breakstat (LexState *ls)'
func (ls *LexState) breakStat() {
	var fs = ls.fs
	var bl = fs.bl
	var upval = false
	for bl != nil && !bl.isBreakable {
		upval = upval || bl.upval
		bl = bl.previous
	}
	if bl == nil {
		ls.xSyntaxError("no loop to break")
	}
	if upval {
		fs.kCodeABC(OP_CLOSE, bl.nActVar, 0, 0)
	}
	fs.kConcat(&bl.breakList, fs.kJump())
}

// 对应C函数：`static void funcstat (LexState *ls, int line)'
func (ls *LexState) funcStat(line int) {
	/* funcstat -> FUNCTION funcname body */
	var v, b = new(expdesc), new(expdesc)
	ls.xNext() /* skip FUNCTION */
	needSelf := ls.funcName(v)
	ls.body(b, needSelf, line)
	ls.fs.kStoreVar(v, b)
	ls.fs.kFixLine(line) /* definition `happens' in the first line */
}

// 对应C函数：`static int funcname (LexState *ls, expdesc *v)'
func (ls *LexState) funcName(v *expdesc) (needSelf bool) {
	/* funcname -> NAME {field} [`:' NAME] */
	ls.singleVar(v)
	for ls.t.token == '.' {
		ls.field(v)
	}
	if ls.t.token == ':' {
		ls.field(v)
		return true
	}
	return false
}

// 对应C函数：`static void localfunc (LexState *ls)'
func (ls *LexState) localFunc() {
	var v, b = new(expdesc), new(expdesc)
	var fs = ls.fs
	ls.newLocalVar(ls.strCheckName(), 0)
	v.initExp(VLOCAL, fs.freeReg)
	fs.kReserveRegs(1)
	ls.adjustLocalVars(1)
	ls.body(b, false, ls.lineNumber)
	fs.kStoreVar(v, b)
	/* debug information will only see the variable after this point! */
	fs.getLocVar(fs.nActVar - 1).startPc = fs.pc
}

// 对应C函数：`static void localstat (LexState *ls)'
func (ls *LexState) localStat() {
	/* stat -> LOCAL NAME {`,' NAME} [`=' explist1] */
	var nVars = 0
	ls.newLocalVar(ls.strCheckName(), nVars)
	nVars++
	for ls.testNext(',') {
		ls.newLocalVar(ls.strCheckName(), nVars)
		nVars++
	}
	var nExps = 0
	var e = new(expdesc)
	if ls.testNext('=') {
		nExps = ls.expList1(e)
	} else {
		e.k = VVOID
		nExps = 0
	}
	ls.adjustAssign(nVars, nExps, e)
	ls.adjustLocalVars(nVars)
}

// 对应C函数：`static void retstat (LexState *ls)'
func (ls *LexState) retStat() {
	/* stat -> RETURN explist */
	var fs = ls.fs
	var first, nRet int /* registers with returned values */

	ls.xNext() /* skip RETURN */
	if blockFollow(ls.t.token) || ls.t.token == ';' {
		/* return no values */
		first = 0
		nRet = 0
	} else {
		var e = new(expdesc)
		nRet = ls.expList1(e) /* optional return values */
		if hasMultiRet(e.k) {
			fs.kSetMultRet(e)
			if e.k == VCALL && nRet == 1 { /* tail call? */
				fs.getCode(e).SetOpCode(OP_TAILCALL)
				LuaAssert(fs.getCode(e).GetArgA() == fs.nActVar)
			}
			first = fs.nActVar
			nRet = LUA_MULTRET /* return all values */
		} else {
			if nRet == 1 { /* only one single value? */
				first = fs.kExp2anyReg(e)
			} else {
				fs.kExp2NextReg(e) /* values must go to the `stack' */
				first = fs.nActVar /* return all `active' values */
				LuaAssert(nRet == fs.freeReg-first)
			}
		}
	}
	fs.kRet(first, nRet)
}

// 对应C函数：`static void exprstat (LexState *ls)'
func (ls *LexState) exprStat() {
	/* stat -> func | assignment */
	var fs = ls.fs
	var v LHSAssign
	ls.primaryExp(&v.v)
	if v.v.k == VCALL { /* stat -> func */
		fs.getCode(&v.v).SetArgC(1) /* call statement uses no results */
	} else { /* stat -> assignment */
		v.prev = nil
		ls.assignment(&v, 1)
	}
}

// LHSAssign structure to chain all variables in the left-hand side of an
// assignment
type LHSAssign struct {
	prev *LHSAssign
	v    expdesc /* variable (global, local, upvalue, or indexed) */
}

// 对应C函数：`static void assignment (LexState *ls, struct LHS_assign *lh, int nvars)'
func (ls *LexState) assignment(lh *LHSAssign, nVars int) {
	ls.checkCondition(VLOCAL <= lh.v.k && lh.v.k <= VINDEXED, "syntax error")
	var e = new(expdesc)
	if ls.testNext(',') { /* assignment -> `,' primaryexp assignment */
		var nv LHSAssign
		nv.prev = lh
		ls.primaryExp(&nv.v)
		if nv.v.k == VLOCAL {
			ls.checkConflict(lh, &nv.v)
		}
		ls.fs.yCheckLimit(nVars, LUAI_MAXCCALLS-ls.L.nCCalls, "variables in assignment")
		ls.assignment(&nv, nVars+1)
	} else { /* assignment -> `=' explist1 */
		ls.checkNextX('=')
		var nExps = ls.expList1(e)
		if nExps != nVars {
			ls.adjustAssign(nVars, nExps, e)
			if nExps > nVars {
				ls.fs.freeReg -= nExps - nVars /* remove extra values */
			}
		} else {
			ls.fs.kSetOneRet(e) /* close last expression */
			ls.fs.kStoreVar(&lh.v, e)
			return /* avoid default */
		}
	}
	e.initExp(VNONRELOC, ls.fs.freeReg-1) /* default assignment */
	ls.fs.kStoreVar(&lh.v, e)
}

// check whether, in an assignment to a local variable, the local variable
// is needed in a previous assignment (to a table). If so, save original
// local value in a safe place and use this safe copy in the previous
// assignment.
// 对应C函数：`static void check_conflict (LexState *ls, struct LHS_assign *lh, expdesc *v)'
func (ls *LexState) checkConflict(lh *LHSAssign, v *expdesc) {
	var fs = ls.fs
	var extra = fs.freeReg /* eventual position to save local variable */
	var conflict = false
	for ; lh != nil; lh = lh.prev {
		if lh.v.k == VINDEXED {
			if lh.v.s.info == v.s.info { /* conflict? */
				conflict = true
				lh.v.s.info = extra /* previous assignment will use safe copy */
			}
			if lh.v.s.aux == v.s.info { /* conflict? */
				conflict = true
				lh.v.s.aux = extra /* previous assignment will use safe copy */
			}
		}
	}
	if conflict {
		fs.kCodeABC(OP_MOVE, fs.freeReg, v.s.info, 0) /* make copy */
		fs.kReserveRegs(1)
	}
}
