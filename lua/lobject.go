package golua

import (
	"bytes"
	"fmt"
)

const (
	// tags for values visible from Lua
	LAST_TAG = LUA_TTHREAD

	NUM_TAGS = LAST_TAG + 1

	// Extra tags for non-values
	LUA_TPROTO   = LAST_TAG + 1
	LUA_TUPVAL   = LAST_TAG + 2
	LUA_TDEADKEY = LAST_TAG + 3
)

/* masks for new-style vararg */
const (
	VARARG_HASARG   = 1
	VARARG_ISVARARG = 2
	VARARG_NEEDSARG = 4
)

type GCHeader = CommonHeader

type StkId = *TValue /* index to stack elements */

type Valuer interface {
	ValuePtr() *Value
	TypePtr() *ttype
}

// SetObj 将obj2的Value和类型赋值给obj1
// 同C `setobj(L,obj1,obj2)`
func SetObj(L *LuaState, obj1, obj2 *TValue) {
	obj1.SetObj(L, obj2)
}

// LuaObjLog2 计算对数
// 对应C函数 `int luaO_log2 (unsigned int x) `
func LuaObjLog2(x uint64) int {
	var Log2 = [256]lu_byte{
		0, 1, 2, 2, 3, 3, 3, 3, 4, 4, 4, 4, 4, 4, 4, 4, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5,
		6, 6, 6, 6, 6, 6, 6, 6, 6, 6, 6, 6, 6, 6, 6, 6, 6, 6, 6, 6, 6, 6, 6, 6, 6, 6, 6, 6, 6, 6, 6, 6,
		7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7,
		7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7,
		8, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8,
		8, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8,
		8, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8,
		8, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8,
	}
	l := -1
	for x >= 256 {
		l += 8
		x >>= 8
	}
	return l + int(Log2[x])
}

func CeilLog2(x uint64) int {
	return LuaObjLog2(x-1) + 1
}

var LuaObjNil = &TValue{tt: LUA_TNIL}

func pushStr(L *LuaState, str []byte) {
	L.Top().SetString(L, L.sNew(str))
	L.IncTop()
}

// this function handles only '%d', '%c', '%f', '%p', and '%s' formats
// 对应C函数：`const char *luaO_pushvfstring (lua_State *L, const char *fmt, va_list argp)'
func oPushVfString(L *LuaState, format []byte, argv ...interface{}) []byte {
	var argi = 0
	var n = 1
	pushStr(L, []byte(""))
	for {
		e := bytes.IndexByte(format, '%')
		if e == -1 {
			break
		}
		L.Top().SetString(L, L.sNewLStr(format[:e]))
		L.IncTop()
		arg := argv[argi]
		argi++
		switch format[e+1] {
		case 's':
			var s []byte
			switch arg.(type) {
			case []byte:
				s = arg.([]byte)
			case string:
				s = []byte(arg.(string))
			default:
				s = []byte("(null)")
			}
			pushStr(L, s)
		case 'c':
			var buff [2]byte
			switch arg.(type) {
			case int32:
				buff[0] = byte(arg.(int32))
			case byte:
				buff[0] = arg.(byte)
			default:
				buff[0] = '?'
			}
			buff[1] = 0
			pushStr(L, buff[:])
		case 'd':
			v := arg.(int)
			L.Top().SetNumber(LuaNumber(v))
			L.IncTop()
		case 'f':
			v := arg.(LuaNumber)
			L.Top().SetNumber(v)
			L.IncTop()
		case 'p':
			buff := fmt.Sprintf("%p", arg)
			pushStr(L, []byte(buff))
		case '%':
			pushStr(L, []byte("%"))
			argi--
		default:
			var buff = [3]byte{
				'%', format[e+1], 0,
			}
			pushStr(L, buff[:])
			argi--
		}
		n += 2
		format = format[e+2:]
	}
	pushStr(L, format)
	L.vConcat(n+1, L.top-L.base-1)
	L.top -= n
	return L.Top().Ptr(-1).StringValue().Bytes
}
