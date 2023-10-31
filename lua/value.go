package golua

import "unsafe"

// Value - Union of all lua values
type Value struct {
	gc GCObject
	p  interface{}
	n  LuaNumber
	b  bool
}

func (v *Value) NumberValue() LuaNumber {
	return v.n
}

func (v *Value) StringValue() *TString {
	return v.gc.ToTString()
}

type lua_TValue struct {
	value Value
	tt
}

type TValue = lua_TValue

// func (v *TValue) PtrAdd(cnt int) *TValue {
// 	p := uintptr(unsafe.Pointer(v)) + unsafe.Sizeof(*v)*uintptr(cnt)
// 	return (*TValue)(unsafe.Pointer(p))
// }

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
	return v.value.gc.ToTString()
}

func (v *TValue) UdataValue() *Udata {
	CheckExp(v.IsUserdata())
	return v.value.gc.ToUdata()
}

// ClosureValue
// 对应C函数：clvalue(o)
func (v *TValue) ClosureValue() Closure {
	CheckExp(v.IsString())
	return v.value.gc.ToClosure()
}

func (v *TValue) TableValue() *Table {
	CheckExp(v.IsBoolean())
	return v.value.gc.ToTable()
}

func (v *TValue) BooleanValue() LuaBoolean {
	CheckExp(v.IsBoolean())
	return v.value.b
}

// IsFalse
// 对应C函数：`l_isfalse(o)'
func (v *TValue) IsFalse() bool {
	return v.IsNil() || (v.IsBoolean() && v.BooleanValue() == false)
}

func (v *TValue) TypePtr() *ttype {
	return &v.tt
}

// 对应C函数：`checkliveness(g,obj)'
func checkliveness(g *GlobalState, obj *TValue) {
	LuaAssert(!obj.IsCollectable() ||
		(obj.Type() == obj.value.gc.Type() && !isdead(g, obj.value.gc)))
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

func (v *TValue) SetAny(x interface{}) {
	v.value.p = x
	v.tt = LUA_TLIGHTUSERDATA
}

func (v *TValue) SetBoolean(x bool) {
	v.value.b = x
	v.tt = LUA_TBOOLEAN
}

func (v *TValue) SetString(L *LuaState, x GCObject) {
	v.value.gc = x
	v.tt = LUA_TSTRING
	checkliveness(L.G(), v)
}

func (v *TValue) SetUserData(L *LuaState, x GCObject) {
	v.value.gc = x
	v.tt = LUA_TUSERDATA
	checkliveness(L.G(), v)
}

func (v *TValue) SetThread(L *LuaState, x GCObject) {
	v.value.gc = x
	v.tt = LUA_TTHREAD
	checkliveness(L.G(), v)
}

func (v *TValue) SetClosure(L *LuaState, x GCObject) {
	v.value.gc = x
	v.tt = LUA_TFUNCTION
	checkliveness(L.G(), v)
}

// SetTable
// 对应C函数：`setthvalue(L,obj,x)'
func (v *TValue) SetTable(L *LuaState, x GCObject) {
	v.value.gc = x
	v.tt = LUA_TTABLE
	checkliveness(L.G(), v)
}

// SetProto
// 对应C函数：`setptvalue(L,obj,x)'
func (v *TValue) SetProto(L *LuaState, x GCObject) {
	v.value.gc = x
	v.tt = LUA_TPROTO
	checkliveness(L.G(), v)
}

func (v *TValue) SetObj(L *LuaState, obj *TValue) {
	v.value = obj.value
	v.tt = obj.tt
	checkliveness(L.G(), v)
}

// 对应C函数：`int luaV_tostring (lua_State *L, StkId obj)'
func (v *TValue) vToString(L *LuaState) bool {
	if !v.IsNumber() {
		return false
	}
	n := v.NumberValue()
	s := NumberToStr(n)
	v.SetString(L, L.sNew([]byte(s)))
	return true
}

// oRawEqualObj 比较两个TValue是否相等
// 对应C函数 `int luaO_rawequalObj (const TValue *t1, const TValue *t2)`
func oRawEqualObj(t1 *TValue, t2 *TValue) bool {
	if t1.Type() != t2.Type() {
		return false
	}
	switch t1.Type() {
	case LUA_TNIL:
		return true
	case LUA_TNUMBER:
		return t1.NumberValue() == t2.NumberValue()
	case LUA_TBOOLEAN:
		return t1.BooleanValue() == t2.BooleanValue()
	case LUA_TLIGHTUSERDATA:
		return t1.PointerValue() == t2.PointerValue()
	default:
		LuaAssert(t1.IsCollectable())
		return t1.GcValue() == t2.GcValue()
	}
}

func (v *TValue) Ptr(n int) StkId {
	p := uintptr(unsafe.Pointer(v))
	if n > 0 {
		p += uintptr(n) * unsafe.Sizeof(TValue{})
	} else {
		p -= uintptr(-n) * unsafe.Sizeof(TValue{})
	}
	return (StkId)(unsafe.Pointer(p))
}
