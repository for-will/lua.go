package golua

import (
	"errors"
	"fmt"
	"os"
	"strings"
)

// LReg 对应C结构：`struct luaL_Reg'
type LReg struct {
	Name string
	Func LuaCFunction
}

// LDoString
// 对应C函数：`luaL_dostring(L, s)'
func (L *LuaState) LDoString(s string) int {
	if r := L.LLoadString(s); r != 0 {
		return r
	}
	return L.PCall(0, LUA_MULTRET, 0)
}

// LLoadString
// 对应C函数：`LUALIB_API int (luaL_loadstring) (lua_State *L, const char *s)'
func (L *LuaState) LLoadString(s string) int {
	return L.LLoadBuffer([]byte(s), s)
}

// LLoadBuffer
// 对应C函数：
// `LUALIB_API int luaL_loadbuffer (lua_State *L, const char *buff, size_t size, const char *name)'
func (L *LuaState) LLoadBuffer(buff []byte, name string) int {
	var ls = &loadS{
		s:    buff,
		size: len(buff),
	}
	return L.Load(getS, ls, []byte(name))
}

// 对应C结构：`struct LoadS'
type loadS struct {
	s    []byte
	size int
}

// 对应C函数：`static const char *getS (lua_State *L, void *ud, size_t *size)'
func getS(L *LuaState, ud interface{}) (buf []byte, size int) {
	var ls = ud.(*loadS)
	_ = L
	if ls.size == 0 {
		return nil, 0
	}
	size = ls.size
	ls.size = 0
	return ls.s, size
}

// LDoFile
// 对应C函数：`luaL_dofile(L, fn)'
func (L *LuaState) LDoFile(filename string) int {
	if ret := L.LLoadFile([]byte(filename)); ret != 0 {
		return ret
	}
	return L.PCall(0, LUA_MULTRET, 0)
}

// LLoadFile
// 对应C函数：`LUALIB_API int luaL_loadfile (lua_State *L, const char *filename)'
func (L *LuaState) LLoadFile(filename []byte) int {
	var lf LoadF
	fNameIndex := L.GetTop() + 1 /* index of filename on the stack */
	lf.extraLine = 0
	if filename == nil {
		L.PushLiteral("=stdin")
		lf.f = STDIN
	} else {
		L.PushFString("@%s", filename)
		var err error
		lf.f, err = fopen(string(filename), os.O_RDONLY)
		if lf.f == nil {
			return errFile(L, "open", fNameIndex, err)
		}
	}
	c, _ := lf.f.getc()
	if c == '#' { /* Unix exec. file? */
		lf.extraLine = 1
		for !lf.f.EOF() && c != '\n' { /* skip first line */
			c, _ = lf.f.getc()
		}
		if c == '\n' {
			c, _ = lf.f.getc()
		}
	}
	if c == LUA_SIGNATURE[0] && len(filename) != 0 { /* binary file？ */
		var err error
		lf.f, err = freopen(string(filename), os.O_RDONLY, lf.f) /* reopen in binary mode */
		if lf.f == nil || err != nil {
			return errFile(L, "reopen", fNameIndex, err)
		}
		/* skip eventual `#!...' */
		for !lf.f.EOF() && c != LUA_SIGNATURE[0] {
			c, _ = lf.f.getc()
		}
		lf.extraLine = 0
	}
	lf.f.ungetc(c)
	status := L.Load(getF, &lf, []byte(L.ToString(-1)))
	readStatus := lf.f.ferror()
	if len(filename) != 0 {
		lf.f.fclose() /* close file (even in case of errors) */
	}
	if readStatus != nil {
		L.SetTop(fNameIndex) /* ignore results from `lua_load' */
		return errFile(L, "read", fNameIndex, readStatus)
	}
	L.Remove(fNameIndex)
	return status
}

// LoadF
// 对应C结构体：`struct LoadF'
type LoadF struct {
	extraLine int
	f         *FILE
	buff      [LUAL_BUFFERSIZE]byte
}

// 对应C函数：`static const char *getF (lua_State *L, void *ud, size_t *size)'
func getF(L *LuaState, ud interface{}) (data []byte, size int) {
	lf := ud.(*LoadF)
	_ = L
	if lf.extraLine != 0 {
		lf.extraLine = 0
		return []byte("\n"), 1
	}
	if lf.f.EOF() {
		return nil, 0
	}
	size = lf.f.fread(lf.buff[:])
	if size > 0 {
		return lf.buff[:], size
	}
	return nil, 0
}

const LUA_ERRFILE = LUA_ERRERR + 1 /* extra error code for `luaL_load' */

// 对应C函数：`static int errfile (lua_State *L, const char *what, int fnameindex)'
func errFile(L *LuaState, what string, fnameIndex int, err error) int {
	filename := L.ToString(fnameIndex)[1:]
	L.PushFString("cannot %s %s: %s", what, filename, err.Error())
	L.Remove(fnameIndex)
	return LUA_ERRFILE
}

// LNewState
// 对应C函数：`LUALIB_API lua_State *luaL_newstate (void)'
func LNewState() *LuaState {
	var l_alloc = func(ud interface{}, ptr interface{}, osize int, nsize int) {

	} /* 对应C函数：`static void *l_alloc (void *ud, void *ptr, size_t osize, size_t nsize) */

	var _panic = func(L *LuaState) int {
		_ = L /* to avoid warnings */
		fmt.Fprintf(os.Stderr, "PANIC: unprotected error in call to Lua API (%s)\n",
			L.ToString(-1))
		return 0
	} /* 对应C函数：`static int panic (lua_State *L)' */

	var L = NewState(l_alloc, nil)
	if L != nil {
		L.AtPanic(_panic)
	}
	return L
}

// LTypeName
// 对应C函数：`luaL_typename(L,i)'
func (L *LuaState) LTypeName(idx int) string {
	return L.TypeName(L.Type(idx))
}

// LRegister
// 对应C函数：`LUALIB_API void (luaL_register) (lua_State *L, const char *libname, const luaL_Reg *l)'
func (L *LuaState) LRegister(libName string, l []LReg) {
	L.IOpenLib(libName, l, 0)
}

// 对应C函数：`static int libsize (const luaL_Reg *l)'
func libSize(l []LReg) int {
	return len(l)
}

// IOpenLib
// 对应C函数：`LUALIB_API void luaI_openlib (lua_State *L, const char *libname, const luaL_Reg *l, int nup)'
func (L *LuaState) IOpenLib(libName string, l []LReg, nup int) {
	if libName != "" {
		var size = libSize(l)
		/* check whether lib already exists */
		L.LFindTable(LUA_REGISTRYINDEX, "_LOADED", 1)
		L.GetField(-1, libName) /* get _LOADED[libName] */
		if !L.IsTable(-1) {     /* not found? */
			L.Pop(1) /* remove previous result */
			/* try global variable (and create one if it does not exist) */
			if L.LFindTable(LUA_GLOBALSINDEX, libName, size) != nil {
				L.LError("name conflict for module '%s'", libName)
			}
			L.PushValue(-1)
			L.SetField(-3, libName) /* _LOADED[libName] = new table */
		}
		L.Remove(-2)         /* remove _LOADED table */
		L.Insert(-(nup + 1)) /* move library table to below upvalues */
	}
	for _, reg := range l {
		for i := 0; i < nup; i++ { /* copy upvalues to the top */
			L.PushValue(-nup)
		}
		L.PushCClosure(reg.Func, nup)
		L.SetField(-(nup + 2), reg.Name)
	}
	L.Pop(nup) /* remove upvalues */
}

// LFindTable
// 对应C函数：`LUALIB_API const char *luaL_findtable (lua_State *L, int idx, const char *fname, int szhint)
func (L *LuaState) LFindTable(idx int, fName string, szHint int) error {
	L.PushValue(idx)
	var e int
	for {
		e = strings.IndexByte(fName, '.')
		if e == -1 {
			e = len(fName)
		}
		L.PushString(fName[:e])
		L.RawGet(-2)
		if L.IsNil(-1) { /* no such field? */
			L.Pop(1) /* remove this nil */
			if e < len(fName) && fName[e] == '.' {
				L.CreateTable(0, 1)
			} else {
				L.CreateTable(0, szHint)
			}
			L.PushString(fName[:e])
			L.PushValue(-2)
			L.SetTable(-4) /* set new table into field */
		} else if !L.IsTable(-1) { /* field has a non-table value? */
			L.Pop(2)                 /* remove table and value */
			return errors.New(fName) /* return problematic part of the name */
		}
		L.Remove(-2) /* remove previous table */
		if e == len(fName) {
			break
		}
		fName = fName[e+1:]
	}
	return nil
}

// LWhere
// 对应C函数：`LUALIB_API void luaL_where (lua_State *L, int level)'
func (L *LuaState) LWhere(level int) {
	// todo: LWhere
	/* ... */
	L.PushLiteral("") /* else, no information available... */
}

// LError
// 对应C函数：`LUALIB_API int luaL_error (lua_State *L, const char *fmt, ...)'
func (L *LuaState) LError(format string, args ...interface{}) int {
	L.LWhere(1)
	L.PushVFString(format, args)
	L.Concat(2)
	return L.Error()
}

// LCheckType
// 对应C函数：`LUALIB_API void luaL_checktype (lua_State *L, int narg, int t)'
func (L *LuaState) LCheckType(nArg int, t ttype) {
	if L.Type(nArg) != t {
		L.tagError(nArg, t)
	}
}

// =======================================================
// Error-report functions
// =======================================================

// LArgError
// 对应C函数：`LUALIB_API int luaL_argerror (lua_State *L, int narg, const char *extramsg)'
func (L *LuaState) LArgError(nArg int, extraMsg string) {
	// todo: LArgError
}

// 对应C函数：`static void tag_error (lua_State *L, int narg, int tag)'
func (L *LuaState) tagError(nArg int, tag ttype) {
	// todo: tagError
}

// LCheckInteger
// 对应C函数：`LUALIB_API lua_Integer luaL_checkinteger (lua_State *L, int narg)'
func (L *LuaState) LCheckInteger(nArg int) LuaInteger {
	var d = L.ToInteger(nArg)
	if d == 0 && !L.IsNumber(nArg) { /* avoid extra test when d is not 0 */
		L.tagError(nArg, LUA_TNUMBER)
	}
	return d
}

// LCheckInt
// 对应C函数：`luaL_checkint(L,n)'
func (L *LuaState) LCheckInt(n int) int {
	return int(L.LCheckInteger(n))
}

// LArgCheck
// 对应C函数：`luaL_argcheck(L, cond,numarg,extramsg)'
func (L *LuaState) LArgCheck(cond bool, numArg int, extraMsg string) {
	if !cond {
		L.LArgError(numArg, extraMsg)
	}
}
