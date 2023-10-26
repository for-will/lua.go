package golua

import "unsafe"

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

func (v *TValue) SetB(L *LuaState, x int) {
	v.value.b = x
	v.tt = LUA_TBOOLEAN
}

func (v *TValue) SetString(L *LuaState, x GCObject) {
	v.value.gc = x
	v.tt = LUA_TSTRING
	// todo: checkliveness(G(L),i_o)
}

func (v *TValue) SetUserData(L *LuaState, x GCObject) {
	v.value.gc = x
	v.tt = LUA_TUSERDATA
	// todo: check live ness
}

func (v *TValue) SetThread(L *LuaState, x GCObject) {
	v.value.gc = x
	v.tt = LUA_TTHREAD
	// todo: check live ness
}

func (v *TValue) SetClosure(L *LuaState, x GCObject) {
	v.value.gc = x
	v.tt = LUA_TFUNCTION
	// todo: check live ness
}

// SetTable
// 对应C函数：`setthvalue(L,obj,x)'
func (v *TValue) SetTable(L *LuaState, x GCObject) {
	v.value.gc = x
	v.tt = LUA_TTABLE
	// todo: check live ness
}

// SetProto
// 对应C函数：`setptvalue(L,obj,x)'
func (v *TValue) SetProto(L *LuaState, x GCObject) {
	v.value.gc = x
	v.tt = LUA_TPROTO
	// todo: check live ness
}

// toString
// 对应C函数：`int luaV_tostring (lua_State *L, StkId obj)'
func (v *TValue) ToString(L *LuaState) bool {
	if !v.IsNumber() {
		return false
	}
	n := v.NumberValue()
	s := NumberToStr(n)
	v.SetString(L, L.sNew([]byte(s)))
	return true
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

func (v *TValue) Ptr(n int) StkId {
	p := uintptr(unsafe.Pointer(v))
	if n > 0 {
		p += uintptr(n) * unsafe.Sizeof(TValue{})
	} else {
		p -= uintptr(-n) * unsafe.Sizeof(TValue{})
	}
	return (StkId)(unsafe.Pointer(p))
}
