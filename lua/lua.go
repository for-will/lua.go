package golua

type ttype int8
type tt = ttype

// 基础数据类型
const (
	LUA_TNONE          ttype = -1
	LUA_TNIL           ttype = 0
	LUA_TBOOLEAN       ttype = 1
	LUA_TLIGHTUSERDATA ttype = 2
	LUA_TNUMBER        ttype = 3
	LUA_TSTRING        ttype = 4
	LUA_TTABLE         ttype = 5
	LUA_TFUNCTION      ttype = 6
	LUA_TUSERDATA      ttype = 7
	LUA_TTHREAD        ttype = 8
)

func (t ttype) Type() ttype {
	return t
}

func (t ttype) IsCollectable() bool {
	return t >= LUA_TSTRING
}

func (t ttype) IsNil() bool {
	return t == LUA_TNIL
}

func (t ttype) IsNumber() bool {
	return t == LUA_TNUMBER
}

func (t ttype) IsString() bool {
	return t == LUA_TSTRING
}

func (t ttype) IsTable() bool {
	return t == LUA_TTABLE
}

func (t ttype) IsFunction() bool {
	return t == LUA_TFUNCTION
}

func (t ttype) IsBoolean() bool {
	return t == LUA_TBOOLEAN
}

func (t ttype) IsUserdata() bool {
	return t == LUA_TUSERDATA
}

func (t ttype) IsThread() bool {
	return t == LUA_TTHREAD
}

func (t ttype) IsLightUserdata() bool {
	return t == LUA_TLIGHTUSERDATA
}

// LuaNumber type of numbers in lua
type LuaNumber = float64

type LuaBoolean = int

// type for integer functions
type lua_Interger = uintptr
