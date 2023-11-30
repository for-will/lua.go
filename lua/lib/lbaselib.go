package lib

import golua "luar/lua"

type (
	LuaState     = golua.LuaState
	LuaCFunction = golua.LuaCFunction
)

func Assert(L *LuaState) int {
	// todo: Assert
	return 0
}

// ipairs
// 对应C函数：`static int luaB_ipairs (lua_State *L)'
func ipairs(L *LuaState) int {
	L.LCheckType(1, golua.LUA_TTABLE)
	L.PushValue(golua.LuaUpValueIndex(1)) /* return generator, */
	L.PushValue(1)                        /* state, */
	L.PushInteger(0)                      /* and initial value */
	return 3
}

// ipairsAux
// 对应C函数：`static int ipairsaux (lua_State *L)'
func ipairsAux(L *LuaState) int {
	var i = L.LCheckInt(2)
	L.LCheckType(1, golua.LUA_TTABLE)
	i++ /*next value */
	L.PushInteger(i)
	L.RawGetI(1, i)
	if L.IsNil(-1) {
		return 0
	} else {
		return 2
	}
}

// 对应C函数：`static int luaB_pairs (lua_State *L)'
func pairs(L *LuaState) int {
	L.LCheckType(1, golua.LUA_TTABLE)
	L.PushValue(golua.LuaUpValueIndex(1)) /* return generator, */
	L.PushValue(1)                        /* state, */
	L.PushNil()                           /* and initial value */
	return 3
}

// 对应C函数：`static int luaB_next (lua_State *L)'
func next(L *LuaState) int {
	L.LCheckType(1, golua.LUA_TTABLE)
	L.SetTop(2) /* create a 2nd argument if there isn't one */
	if L.LuaNext(1) {
		return 2
	} else {
		L.PushNil()
		return 1
	}
}

// 对应C函数：`static int luaB_newproxy (lua_State *L) '
func newProxy(L *LuaState) int {
	L.SetTop(1)
	L.NewUserData(0) /* create proxy */
	if L.ToBoolean(1) == false {
		return 1 /* no metatable */
	} else if L.IsBoolean(1) {
		L.NewTable()    /* create a new metatable `m' ... */
		L.PushValue(-1) /* ... and mark `m' as a valid metatable */
		L.PushBoolean(true)
		L.RawSet(golua.LuaUpValueIndex(1)) /* weaktable[m] = true */
	} else {
		var validProxy bool /* to check if weaktable[metatalbe(u)] == true */
		if L.GetMetaTable(1) != 0 {
			L.RawGet(golua.LuaUpValueIndex(1))
			validProxy = L.ToBoolean(-1)
			L.Pop(1) /* remove value */
		}
		L.LArgCheck(validProxy, 1, "boolean or proxy expected")
		L.GetMetaTable(1) /* metatable is valid; get it */
	}
	L.SetMetaTable(2)
	return 1
}

var coFuncs = []golua.LReg{}

var baseFuncs = []golua.LReg{
	{"assert", Assert},
}

// 对应C函数：`static void auxopen (lua_State *L, const char *name, lua_CFunction f, lua_CFunction u)'
func auxOpen(L *LuaState, name string, f LuaCFunction, u LuaCFunction) {
	L.PushCClosure(u, 0)
	L.PushCClosure(f, 1)
	L.SetField(-2, name)
}

func baseOpen(L *golua.LuaState) {
	/* set global _G */
	L.PushValue(golua.LUA_GLOBALSINDEX)
	L.SetGlobal("_G")
	/* open lib into global table */
	L.LRegister("_G", baseFuncs)
	L.PushLiteral(golua.LUA_VERSION)
	L.SetGlobal("_VERSION") /* set global _VERSION */
	/* `ipairs' and `pairs' need auxiliary functions as upvalues */
	auxOpen(L, "ipairs", ipairs, ipairsAux)
	auxOpen(L, "pairs", pairs, next)
	/* `newproxy' needs a weaktable as upvalue */
	L.CreateTable(0, 1) /* new table `w' */
	L.PushValue(-1)     /*`w' will be its own metatable */
	L.SetMetaTable(-2)
	L.PushLiteral("kv")
	L.SetField(-2, "__mode") /* metatable(w).__mode = "kv" */
	L.PushCClosure(newProxy, 1)
	L.SetGlobal("newproxy") /* set global `newproxy' */
}

// LuaOpenBase
// 对应C函数：`LUALIB_API int luaopen_base (lua_State *L)'
func LuaOpenBase(L *LuaState) int {
	baseOpen(L)
	L.LRegister(LUA_COLIBNAME, coFuncs)
	return 2
}
