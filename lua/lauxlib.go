package golua

import (
	"fmt"
	"os"
)

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
