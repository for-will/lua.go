package golua

const (
	LUA_VERSION     = "Lua 5.1"
	LUA_RELEASE     = "Lua 5.1.4"
	LUA_VERSION_NUM = 501
	LUA_COPYRIGHT   = "Copyright (C) 1994-2008 Lua.org, PUC-Rio"
	LUA_AUTHORS     = "R. Ierusalimschy, L. H. de Figueiredo & W. Celes"
)

const LUA_SIGNATURE = "\033Lua" /* mark for precompiled code (`<esc>Lua') */

const LUA_MULTRET = -1 /* option for multiple returns in `lua_pcall' and `lua_call' */

/* pseudo-indices */
const (
	LUA_REGISTRYINDEX = -10000
	LUA_ENVIRONINDEX  = -10001
	LUA_GLOBALSINDEX  = -10002
)

/* thread status; 0 is OK */
const (
	LUA_YIELD     = 1
	LUA_ERRRUN    = 2
	LUA_ERRSYNTAX = 3
	LUA_ERRMEM    = 4
	LUA_ERRERR    = 5
)

// LuaUpValueIndex
// 对应C函数：`lua_upvalueindex(i)'
func LuaUpValueIndex(i int) int {
	return LUA_GLOBALSINDEX - i
}

// LUA_MINSTACK
/* minimum Lua stack available to a C function */
const LUA_MINSTACK = 20

/* functions that read/write blocks when loading/dumping Lua chunks */

// LuaReadFunc
// 对应C：`typedef const char * (*lua_Reader) (lua_State *L, void *ud, size_t *sz)'
type LuaReadFunc func(L *LuaState, ud interface{}) (buf []byte, size int)

// LuaWriteFunc
// 对应C：`typedef int (*lua_Writer) (lua_State *L, const void* p, size_t sz, void* ud)'
type LuaWriteFunc func(L *LuaState, p []byte, sz int, ud interface{})

// LuaAlloc
// prototype for memory-allocation functions
// 对应C：`typedef void * (*lua_Alloc) (void *ud, void *ptr, size_t osize, size_t nsize)'
type LuaAlloc func(ud interface{}, ptr interface{}, osize int, nsize int)

// LuaNumber type of numbers in lua
type LuaNumber = float64

type LuaBoolean = bool

// LuaInteger type for integer functions
type LuaInteger = int

// ToString
// 对应C函数：`lua_tostring(L,i)'
func (L *LuaState) ToString(i int) string {
	s, _ := L.ToLString(i)
	return string(s)
}

//
// Debug API
//

/* Event codes */
const (
	LUA_HOOKCALL    = 0
	LUA_HOOKRET     = 1
	LUA_HOOKLINE    = 2
	LUA_HOOKCOUNT   = 3
	LUA_HOOKTAILRET = 4
)

/* Event masks */
const (
	LUA_MASKCALL  = 1 << LUA_HOOKCALL
	LUA_MASKRET   = 1 << LUA_HOOKRET
	LUA_MASKLINE  = 1 << LUA_HOOKLINE
	LUA_MASKCOUNT = 1 << LUA_HOOKCOUNT
)

// LuaHook
// Function to be called by the debuger in specific events
// 对应C类型：`typedef void (*lua_Hook) (lua_State *L, lua_Debug *ar)'
type LuaHook func(L *LuaState, ar *LuaDebug)

// LuaDebug
// 对应C结构体：`struct lua_Debug'
type LuaDebug struct {
	Event       int
	Name        string /* (n) */
	NameWhat    string /* (n) `global', `local', `field', `method' */
	What        string /* (S) `Lua', `C', `main', `tail' */
	Source      string /* (S) */
	CurrentLine int    /* (l) */
	NUps        int    /* (u) number of upvalues */
	LineDefined int    /* (S) */
	ShortSrc    []byte /* (S) */
	/* private part */
	iCI int /* active function */
}

// LuaOpen
// 对应C函数：`lua_open()'
func LuaOpen() *LuaState {
	return LNewState()
}

// Register
// 对应C函数：`lua_register(L,n,f)'
func (L *LuaState) Register(name string, f LuaCFunction) {
	L.PushCFunction(f)
	L.SetGlobal(name)
}

// PushCFunction
// 对应C函数：`lua_pushcfunction(L,f)'
func (L *LuaState) PushCFunction(f LuaCFunction) {
	L.PushCClosure(f, 0)
}

// SetGlobal
// 对应C函数：`lua_setglobal(L,s)'
func (L *LuaState) SetGlobal(k string) {
	L.SetField(LUA_GLOBALSINDEX, k)
}

// Pop
// 对应C函数：`lua_pop(L,n)'
func (L *LuaState) Pop(n int) {
	L.SetTop(-n - 1)
}
