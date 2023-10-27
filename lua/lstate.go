package golua

const (
	EXTRA_STACK = 5 /* extra stack space to handle TM calls and some other extras */
)

// GCObject Union of all collectable objects
// type GCObject interface{}

type GCObject interface {
	Type() ttype
	Next() GCObject
	SetNext(obj GCObject)
	ToString() *TString // gco2ts
	ToTable() *Table
	ToClosure() Closure
	ToUpval() *UpVal
	ToUdata() *Udata
}

type GlobalState struct {
	StrT      *StringTable     /* hash table for strings */
	buff      MBuffer          /* temporary buffer for string concatentation */
	lRegistry TValue           /* */
	mt        [NUM_TAGS]*Table /* metatables for basic types */
	tmname    [TM_N]*TString   /* array with tag-method names */
}

func (g *GlobalState) LuaCWhite() lu_byte {
	// todo:
	return 0
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
	lGt           TValue       /* table of globals */
	env           TValue       /* temporary place for environments */
	openUpval     GCObject     /* list of open upvalues in this stack */
	gcList        GCObject     /* */
	errorJmp      *LuaLongJmp  /* current error recover point */
	errFunc       int          /* current error handling function (stack index) */
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
	L.correctStack(oldStack)
}

// 对应C函数：`static void correctstack (lua_State *L, TValue *oldstack)`
func (L *LuaState) correctStack(oldStack []TValue) {
	// todo:
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
