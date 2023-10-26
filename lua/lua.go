package golua

const LUA_SIGNATURE = "\033Lua"

/* option for multiple returns in `lua_pcall' and `lua_call' */
const LUA_MULTRET = -1

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

/* functions that read/write blocks when loading/dumping Lua chunks */

// LuaReadFunc
// 对应C：`typedef const char * (*lua_Reader) (lua_State *L, void *ud, size_t *sz)'
type LuaReadFunc func(L *LuaState, ud interface{}) (buf []byte, size int)

// LuaWriteFunc
// 对应C：`typedef int (*lua_Writer) (lua_State *L, const void* p, size_t sz, void* ud)'
type LuaWriteFunc func(L *LuaState, p []byte, sz int, ud interface{})

// LuaNumber type of numbers in lua
type LuaNumber = float64

type LuaBoolean = int

// type for integer functions
type lua_Interger = uintptr

// ToString
// 对应C函数：`lua_tostring(L,i)'
func (L *LuaState) ToString(i int) []byte {
	s, _ := L.ToLString(i)
	return s
}
