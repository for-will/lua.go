package golua

import "unsafe"

const (
	EXTRA_STACK      = 5 /* extra stack space to handle TM calls and some other extras */
	BASIC_CI_SIZE    = 8
	BASIC_STACK_SIZE = 2 * LUA_MINSTACK
)

// GCObject Union of all collectable objects
// type GCObject interface{}

type GCObject interface {
	gcType() ttype
	setType(t ttype)
	Next() GCObject
	SetNext(obj GCObject)
	SetMarked(m lu_byte)
	ToTString() *TString // gco2ts
	ToTable() *Table
	ToClosure() Closure
	ToUpval() *UpVal
	ToUdata() *Udata
	ToThread() *LuaState
}

// GlobalState
// `global state', shared by all threads of this state
type GlobalState struct {
	StrT         *StringTable     /* hash table for strings */
	freeAlloc    LuaAlloc         /* function to reallocate memory */
	ud           interface{}      /* auxiliary data to `frealloc' */
	currentWhite lu_byte          /* */
	gcState      lu_byte          /* state of garbage collector */
	sweepStrGC   int              /* position of sweep in `strt' */
	rootGC       GCObject         /* list of all collectable objects */
	sweepGc      *GCObject        /* position of sweep in `rootgc' */
	gray         []GCObject       /* list of gray objects */
	grayAgain    []GCObject       /* list of objects to be traversed atomically */
	weak         []GCObject       /* list of weak tables (to be cleared) */
	tmUData      GCObject         /* last element of list of userdata to be GC */
	buff         MBuffer          /* temporary buffer for string concatentation */
	GCThreshold  lu_mem           /* */
	totalBytes   lu_mem           /* number of bytes currently allocated */
	estimate     lu_mem           /* an estimate of number of bytes actually in use */
	gcDept       lu_mem           /* how much GC is `behind schedule' */
	gcPause      int              /* size of pause between successive GCs */
	gcStepMul    int              /* GC `granularity' */
	panic        LuaCFunction     /* to be called in unprotected errors */
	lRegistry    TValue           /* */
	mainThread   *LuaState        /* */
	uvHead       UpVal            /* head of double-linked list of all open upvalues */
	mt           [NUM_TAGS]*Table /* metatables for basic types */
	tmName       [TM_N]*TString   /* array with tag-method names */
}

// 对应C函数：`luaC_white(g)'
func (g *GlobalState) cWhite() lu_byte {
	return g.currentWhite & WHITEBITS
}

type LuaState struct {
	CommonHeader
	status        lu_byte
	top           int          /* first free slot in the stack */
	base          int          /* base of current function */
	lG            *GlobalState /* */
	ci            int          /* call info for current function */
	savedPc       *Instruction /* `savedpc' of current function */
	stackLast     int          /* last free slot in the stack */
	stack         []TValue     /* stack base */
	endCi         int          /* points after end of ci array */
	baseCi        []CallInfo   /* array of CallInfo's */
	stackSize     int          /* */
	sizeCi        int          /* size of array `base_ci' */
	nCCalls       int          /* number of nested C calls */
	baseCCalls    int          /* nested C calls when resuming coroutine */
	hookMask      lu_byte      /* */
	allowHook     lu_byte      /* */
	baseHootCount int          /* */
	hookCount     int          /* */
	hook          LuaHook      /* */
	lGt           TValue       /* table of globals */
	env           TValue       /* temporary place for environments */
	openUpval     GCObject     /* list of open upvalues in this stack */
	gcList        GCObject     /* */
	errorJmp      *LuaLongJmp  /* current error recover point */
	errFunc       int          /* current error handling function (stack index) */
}

// LG
// Main  thread combines a thread state and the global state
// 对应C结构体：`struct LG'
type LG struct {
	l LuaState
	g GlobalState
}

// NewState
// 对应C函数：`LUA_API lua_State *lua_newstate (lua_Alloc f, void *ud)'
func NewState(f LuaAlloc, ud interface{}) *LuaState {
	l := &LG{
		g: GlobalState{
			StrT: new(StringTable),
		},
	}
	L := &l.l
	g := &l.g
	L.next = nil
	L.tt = LUA_TTHREAD
	g.currentWhite = 1<<WHITEBITS | 1<<FIXEDBIT
	L.marked = g.cWhite()
	L.marked |= 1<<FIXEDBIT | 1<<SFIXEDBIT
	preinit_state(L, g)
	g.freeAlloc = f
	g.ud = ud
	g.mainThread = L
	g.uvHead.l.prev = &g.uvHead
	g.uvHead.l.next = &g.uvHead
	g.GCThreshold = 0 /* mark it as unfinished state */
	g.StrT.Size = 0
	g.StrT.NrUse = 0
	g.StrT.Hash = nil
	L.Registry().SetNil()
	g.buff.Init()
	g.panic = nil
	g.gcState = GCSPause
	g.rootGC = L
	g.sweepStrGC = 0
	g.sweepGc = &g.rootGC
	g.gray = nil
	g.grayAgain = nil
	g.weak = nil
	g.tmUData = nil
	g.totalBytes = int(unsafe.Sizeof(LG{}))
	g.gcPause = LUAI_GCPAUSE
	g.gcStepMul = LUAI_GCMUL
	g.gcStepMul = 0
	for i := 0; i < int(NUM_TAGS); i++ {
		g.mt[i] = nil
	}
	if L.dRawRunProtected(f_luaopen, nil) != 0 {
		/* memory allocation error: free partial state */
		L.closeState()
		L = nil
	} else {
		LUAIUserStateOpen(L)
	}
	return L

}

// 对应C函数：`static void close_state (lua_State *L)'
func (L *LuaState) closeState() {
	g := L.G()
	L.fClose(&L.stack[0]) /* close all upvalues for this thread */
	L.cFreeAll()
	// todo: LuaAssert(g.rootGC == L) /* collect all objects */
	// todo: LuaAssert(g.StrT.NUse == 0)
	L.G().StrT.Hash = nil
	g.buff.Free()
	freestack(L, L)
	// todo: LuaAssert(g.totalBytes == int(unsafe.Sizeof(LG{})))
	g.ud = nil
}

// 对应C函数：`static void stack_init (lua_State *L1, lua_State *L)'
func stack_init(L1 *LuaState, L *LuaState) {
	/* initialize CallInfo array*/
	L1.baseCi = make([]CallInfo, BASIC_CI_SIZE)
	L1.ci = 0
	L1.sizeCi = BASIC_CI_SIZE
	L1.endCi = L1.sizeCi - 1
	/* initialize stack array */
	L1.stack = make([]TValue, BASIC_STACK_SIZE+EXTRA_STACK)
	L1.stackSize = BASIC_STACK_SIZE + EXTRA_STACK
	L1.top = 0
	L1.stackLast = L1.stackSize - EXTRA_STACK - 1
	/* initialize first ci */
	L1.CI().fn = L1.Top()
	L1.Top().SetNil() /* `function' entry for this `ci' */
	L1.top++
	L1.base = L1.top
	L1.CI().base = L1.top
	L1.CI().top = L1.top + LUA_MINSTACK
}

// 对应C函数：static void freestack (lua_State *L, lua_State *L1)
func freestack(L *LuaState, L1 *LuaState) {
	L1.baseCi = nil
	L1.stack = nil
}

// open parts that may cause memory-allocation errors
// 对应C函数：`static void f_luaopen (lua_State *L, void *ud)'
func f_luaopen(L *LuaState, ud interface{}) {
	g := L.G()
	_ = ud
	stack_init(L, L)                          /* init stack */
	L.GlobalTable().SetTable(L, L.hNew(0, 2)) /* table of globals */
	L.Registry().SetTable(L, L.hNew(0, 2))    /* registry */
	L.sResize(MINSTRTABSIZE)                  /* initial size of string table */
	L.tInit()
	L.xInit()
	L.sNewLiteral(MEMERRMSG).Fix()
	g.GCThreshold = 4 * g.totalBytes
}

// 对应C函数：`static void preinit_state (lua_State *L, global_State *g)'
func preinit_state(L *LuaState, g *GlobalState) {
	L.lG = g
	L.stack = nil
	L.stackSize = 0
	L.errorJmp = nil
	L.hook = nil
	L.hookMask = 0
	L.baseHootCount = 0
	L.allowHook = 1
	ResetHookCount(L)
	L.openUpval = nil
	L.sizeCi = 0
	L.nCCalls = 0
	L.baseCCalls = 0
	L.status = 0
	L.baseCi = nil
	L.ci = 0
	L.savedPc = nil
	L.errFunc = 0
	L.GlobalTable().SetNil()
}

func (L *LuaState) G() *GlobalState {
	return L.lG
}

// Lock 什么也不做
// 对应C：lua_lock(L)
func (L *LuaState) Lock() {

}

// Unlock 什么也不做
// 对应C：lua_unlock(L)
func (L *LuaState) Unlock() {

}

// GlobalTable table of globals
func (L *LuaState) GlobalTable() *TValue {
	return &L.lGt
}

// Registry
// 对应C函数：`registry(L)'
func (L *LuaState) Registry() *TValue {
	return &L.G().lRegistry
}

func (L *LuaState) Top() StkId {
	return &L.stack[L.top]
}

func (L *LuaState) Base() StkId {
	return &L.stack[L.base]
}

// AtTop 返回相对于top距离offset个元素的栈上成员指针
func (L *LuaState) AtTop(offset int) StkId {
	return &L.stack[L.top+offset]
}

// AtBase 返回相对于base距离offset个元素的栈上成员指针
func (L *LuaState) AtBase(offset int) StkId {
	return &L.stack[L.base+offset]
}

// GetTop
// 对应C函数：`LUA_API int lua_gettop (lua_State *L)'
func (L *LuaState) GetTop() int {
	return L.top - L.base
}

// SetTop
// 对应C函数：`LUA_API void lua_settop (lua_State *L, int idx)'
func (L *LuaState) SetTop(idx int) {
	L.Lock()
	if idx >= 0 {
		ApiCheck(L, idx <= L.stackLast-L.base)
		for L.top < L.base+idx {
			L.stack[L.top].SetNil()
			L.top++
		}
		L.top = L.base + idx
	} else {
		ApiCheck(L, -(idx+1) <= (L.top-L.base))
		L.top += idx + 1 /* `subtract' index (index is negative) */
	}
	L.Unlock()
}

// IncTop
// 对应C函数：`incr_top(L)'
func (L *LuaState) IncTop() {
	L.dCheckStack(1)
	L.top++
}

// dCheckStack
// 对应C函数：`luaD_checkstack(L,n)'
func (L *LuaState) dCheckStack(n int) {
	if L.stackLast-L.top <= n {
		L.dGrowStack(n)
	} else if CondHardStackTests() {
		L.dReAllocStack(L.stackSize - EXTRA_STACK - 1)
	}
}

// dGrowStack
// 对应C函数：`void luaD_growstack (lua_State *L, int n)'
func (L *LuaState) dGrowStack(n int) {
	if n <= L.stackSize { /* double size is enough? */
		L.dReAllocStack(2 * L.stackSize)
	} else {
		L.dReAllocStack(L.stackLast + n)
	}
}

// dReAllocStack
// 对应C函数：`void luaD_reallocstack (lua_State *L, int newsize)'
func (L *LuaState) dReAllocStack(newSize int) {
	oldStack := L.stack
	realSize := newSize + 1 + EXTRA_STACK
	LuaAssert(L.stackLast == L.stackSize-EXTRA_STACK-1)
	newStack := make([]TValue, realSize)
	copy(newStack, oldStack)
	L.stack = newStack
	L.stackSize = realSize
	L.stackLast = newSize
	correctstack(L, oldStack)
}

// 对应C函数：`static void correctstack (lua_State *L, TValue *oldstack)`
func correctstack(L *LuaState, oldStack []TValue) {
	// L.top，L.base 已经是在stack中的下标，不用处理
	stackPtr := uintptr(unsafe.Pointer(&L.stack[0]))
	oldPtr := uintptr(unsafe.Pointer(&oldStack[0]))
	correct := func(v *TValue) *TValue {
		if v == nil {
			return nil
		}
		p := uintptr(unsafe.Pointer(v)) - oldPtr + stackPtr
		return (*TValue)(unsafe.Pointer(p))
	}
	for up := L.openUpval; up != nil; up = up.Next() {
		uv := up.ToUpval()
		uv.v = correct(uv.v)
	}
	for i := 0; i <= L.ci; i++ {
		ci := L.baseCi[i]
		ci.fn = correct(ci.fn)
		// ci.base和ci.top不需要处理
	}
}

// CurrFunc
// 对应C函数：`curr_func(L)'
func (L *LuaState) CurrFunc() Closure {
	return L.CI().fn.ClosureValue()
}

func (L *LuaState) CI() *CallInfo {
	return &L.baseCi[L.ci]
}

//
// 添加的一些函数，简化操作
//

func (L *LuaState) PushObj(obj *TValue) {
	SetObj(L, L.Top(), obj)
	L.top++
}

func (L *LuaState) PushTable(h *Table) {
	L.Top().SetTable(L, h)
	L.top++
}

// DecrHookCount 相当于C语言中的`--L.hookCount'
func (L *LuaState) DecrHookCount() int {
	L.hookCount--
	return L.hookCount
}

// CallInfo
// information about a call.
// 对应C函数：`struct CallInfo '
type CallInfo struct {
	base      int   /* base for this function */
	fn        StkId /* function index in the stack ; 因为func在go中为关键字，改名为fn */
	top       int   /* top for this function */
	savedPc   *Instruction
	nResults  int /* expected number of results from this function */
	tailCalls int /* number of tail calls lost under this entry */
}

// Func
// 对应C函数：`ci_func(ci)'
func (ci *CallInfo) Func() Closure {
	return ci.fn.ClosureValue()
}

// 对应C函数：`f_isLua(ci)'
func (ci *CallInfo) fIsLua() bool {
	return ci.Func().IsLFunction()
}

// IsLua
// 对应C函数：`isLua(ci)'
func (ci *CallInfo) IsLua() bool {
	return ci.fn.IsFunction() && ci.fIsLua()
}

// Close
// 对应C函数：`LUA_API void lua_close (lua_State *L)'
func (L *LuaState) Close() {
	L = L.G().mainThread /* only the main thread can be closed */
	L.Lock()
	L.fClose(&L.stack[0]) /* close all upvalues for this thread */
	// todo:  L.cSepareateUdata(1)  /* separate udata that have GC metamethods */
	L.errFunc = 0 /* no error function during GC metamethods */

	for { /* repeat until no more errors */
		L.ci = 0
		L.base = L.CI().base
		L.top = L.base
		L.nCCalls = 0
		L.baseCCalls = 0
		if L.dRawRunProtected(callAllGcTM, nil) == 0 {
			break
		}
	}
	LuaAssert(L.G().tmUData == nil)
	LUAIUserStateClose(L)
	L.closeState()
}

// 对应C函数：`static void callallgcTM (lua_State *L, void *ud)'
func callAllGcTM(L *LuaState, ud interface{}) {
	// todo: callAllGcTM
}
