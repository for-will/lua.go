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
	nActVar    lu_byte                     /* number of active local variables */
	upvalues   [LUAI_MAXUPVALUES]upvaldesc /* upvalues */
	actvar     [LUAI_MAXVARS]uint16        /* declared-variable stack */
}

// BlockCnt
// nodes for block list (list of active blocks)
// 对应C结构：`struct BlockCnt'
type BlockCnt struct {
	previous    *BlockCnt /* chain */
	breakList   int       /* list of jumps out of this loop */
	nActVar     lu_byte   /* # active locals outside the breakable structure */
	upval       lu_byte   /* true if some variable in the block is an up-value */
	isBreakable lu_byte   /* true if `block' is a loop */
}

// 对应C结构：`struct upvaldesc'
type upvaldesc struct {
	k    lu_byte
	info lu_byte
}

// YParser
// 对应C函数：`Proto *luaY_parser (lua_State *L, ZIO *z, Mbuffer *buff, const char *name)'
func (L *LuaState) YParser(z *ZIO, buff *MBuffer, name []byte) *Proto {
	var lexState LexState
	var funcState FuncState
	lexState.buff = buff
	xSetInput(L, &lexState, z, L.sNew(name))
	openFunc(&lexState, &funcState)
	funcState.f.isVarArg = VARARG_ISVARARG /* main func. is always vararg */

	return nil
}

// 对应C函数：`static void openFunc (LexState *ls, FuncState *fs)'
func openFunc(ls *LexState, fs *FuncState) {
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
