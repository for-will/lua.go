package golua

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"
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

// converts an integer to a "floating point byte", represented as
// (eeeeexxx), where the real value is (1xxx) * 2^(eeeee - 1) if
// eeeee != 0 and (xxx) otherwise.
// 对应C函数：`int luaO_int2fb (unsigned int x)'
func oInt2Fb(x uint) int {
	var e = 0 /* expoent */
	for x >= 16 {
		x = (x + 1) >> 1
		e++
	}
	if x < 8 {
		return int(x)
	} else {
		return (e+1)<<3 | int(x-8)
	}
}

// converts back
// 对应C函数：`int luaO_fb2int (int x)'
func oFb2Int(x int) int {
	var e = (x >> 3) & 31
	if e == 0 {
		return x
	} else {
		return ((x & 7) + 8) << (e - 1)
	}
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
func (L *LuaState) oPushVfString(format []byte, argv []interface{}) []byte {
	var argi = 0
	var n = 1
	pushStr(L, []byte(""))
	for {
		e := bytes.IndexByte(format, '%')
		if e == -1 {
			break
		}
		L.Top().SetString(L, L.sNewStr(format[:e]))
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

// 对应C函数：`const char *luaO_pushfstring (lua_State *L, const char *fmt, ...)'
func (L *LuaState) oPushFString(format string, args ...interface{}) []byte {
	return L.oPushVfString([]byte(format), args)
}

// 对应C函数：`int luaO_str2d (const char *s, lua_Number *result)'
func oStr2d(s string, result *LuaNumber) (ok bool) {
	s = strings.TrimRight(s, " \t\n\r")
	if s[len(s)-1] == 'x' || s[len(s)-1] == 'X' {
		i, err := strconv.ParseInt(s[:len(s)-1], 16, 64)
		if err != nil {
			return false
		}
		*result = LuaNumber(i)
		return true
	}

	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return false
	}
	*result = v
	return true
}

// 对应C函数：`void luaO_chunkid (char *out, const char *source, size_t bufflen)'
func oChunkId(out []byte, source string, bufflen int) {
	if source[0] == '=' {
		copy(out, source[1:]) /* remove first char*/
		// out[len(out)-1] = 0   /* ensures null termination */
	} else { /* out = "source", or "...source" */
		if source[0] == '@' {
			source = source[1:] /* skip the `@' */
			bufflen -= len(" '...' ")
			var l = len(source)
			if l > bufflen {
				source = source[l-bufflen:] /* get last part of file name */
				copy(out, "...")
				out = out[3:]
			}
			copy(out, source)
		} else { /* out = [string "string"] */
			var l = strings.IndexAny(source, "\n\r") /* stop at first newline */
			if l < 0 {
				l = len(source)
			}
			bufflen -= len(" [string \"...\"] ")
			out = out[:0]
			out = append(out, []byte("[string \"")...)
			if l > bufflen {
				l = bufflen
			}
			if l != len(source) { /* must truncate? */
				out = append(out, source[:l]...)
				out = append(out, "..."...)
			} else {
				out = append(out, source...)
			}
			out = append(out, "\"]"...)
		}
	}
}
