package golua

import "unsafe"

const (
	// tags for values visible from Lua
	LAST_TAG = LUA_TTHREAD

	NUM_TAGS = LAST_TAG + 1

	// Extra tags for non-values
	LUA_TPROTO   = LAST_TAG + 1
	LUA_TUPVAL   = LAST_TAG + 2
	LUA_TDEADKEY = LAST_TAG + 3
)

type GCHeader = CommonHeader

type StkId = *TValue /* index to stack elements */

// Value - Union of all lua values
type Value struct {
	gc GCObject
	p  interface{}
	n  LuaNumber
	b  int
}

func (v *Value) NumberValue() LuaNumber {
	return v.n
}

func (v *Value) StringValue() *TString {
	return v.gc.ToString()
}

type lua_TValue struct {
	value Value
	tt
}

type TValue = lua_TValue

func (v *TValue) PtrAdd(cnt int) *TValue {
	p := uintptr(unsafe.Pointer(v)) + unsafe.Sizeof(*v)*uintptr(cnt)
	return (*TValue)(unsafe.Pointer(p))
}

func (v *TValue) ValuePtr() *Value {
	return &v.value
}

func (v *TValue) GcValue() GCObject {
	CheckExp(v.IsCollectable())
	return v.value.gc
}

func (v *TValue) PointerValue() interface{} {
	CheckExp(v.IsLightUserdata())
	return v.value.p
}

func (v *TValue) NumberValue() LuaNumber {
	CheckExp(v.IsNumber())
	return v.value.n
}

func (v *TValue) StringValue() *TString {
	CheckExp(v.IsString())
	return v.value.gc.ToString()
}

func (v *TValue) BooleanValue() LuaBoolean {
	CheckExp(v.IsBoolean())
	return v.value.b
}

func (v *TValue) TypePtr() *ttype {
	return &v.tt
}

// SetNil 将v赋值为nil
// 对应C函数 `setnilvalue(obj)`
func (v *TValue) SetNil() {
	v.value.gc = nil
	v.value.p = nil
	v.tt = LUA_TNIL
}

// SetNumber 将v赋值为数字x
// 对应C函数 `setnvalue(obj,x)`
func (v *TValue) SetNumber(x LuaNumber) {
	v.value.gc = nil
	v.value.p = nil
	v.value.n = x
	v.tt = LUA_TNUMBER
}

func (v *TValue) SetP(x interface{}) {
	v.value.p = x
	v.tt = LUA_TLIGHTUSERDATA
}

func (v *TValue) SetB(x int) {
	v.value.b = x
	v.tt = LUA_TBOOLEAN
}

func (v *TValue) SetS(x GCObject) {
	v.value.gc = x
	v.tt = LUA_TSTRING
	// todo: check live ness
}

func (v *TValue) SetU(x GCObject) {
	v.value.gc = x
	v.tt = LUA_TUSERDATA
	// todo: check live ness
}

func (v *TValue) SetTh(x GCObject) {
	v.value.gc = x
	v.tt = LUA_TTHREAD
	// todo: check live ness
}

func (v *TValue) SetCl(x GCObject) {
	v.value.gc = x
	v.tt = LUA_TFUNCTION
	// todo: check live ness
}

func (v *TValue) SetH(x GCObject) {
	v.value.gc = x
	v.tt = LUA_TTABLE
	// todo: check live ness
}

func (v *TValue) SetPt(x GCObject) {
	v.value.gc = x
	v.tt = LUA_TTABLE
	// todo: check live ness
}

// IsEqualTo 比较两个TValue是否相等
// 对应C函数 `int luaO_rawequalObj (const TValue *t1, const TValue *t2)`
func (v *TValue) IsEqualTo(t *TValue) bool {
	if v.Type() != t.Type() {
		return false
	}
	switch v.Type() {
	case LUA_TNIL:
		return true
	case LUA_TNUMBER:
		return v.NumberValue() == t.NumberValue()
	case LUA_TBOOLEAN:
		return v.BooleanValue() == t.BooleanValue()
	case LUA_TLIGHTUSERDATA:
		return v.PointerValue() == t.PointerValue()
	default:
		LuaAssert(v.IsCollectable())
		return v.GcValue() == t.GcValue()
	}
}

type Valuer interface {
	ValuePtr() *Value
	TypePtr() *ttype
}

// SetObj 将obj2的Value和类型赋值给obj1
// 同C `setobj(L,obj1,obj2)`
func SetObj(L *LuaState, obj1, obj2 Valuer) {
	*obj1.ValuePtr() = *obj2.ValuePtr()
	*obj1.TypePtr() = *obj2.TypePtr()
	// todo: checkliveness(G(L),o1)
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
