package golua

import (
	"testing"
)

func TestCommonHeader(t *testing.T) {
	t.Log(&CommonHeader{})
	// var node Node
	// var tb Table
	v1 := &TValue{
		value: Value{
			n: 199,
		},
		tt: LUA_TNUMBER,
	}
	v2 := &TValue{}
	SetObj(nil, v2, v1)
	t.Log(*v2)
}

func TestAnyCompare(t *testing.T) {
	v1 := &TString{}
	var p1 GCObject = v1
	var p2 GCObject = v1
	var p0 interface{} = v1
	p2 = nil
	t.Log(p1 == p2, p2 == p0, p0 == p1)

}

func TestSliceCopy(t *testing.T) {

	a := []int{1, 2, 3, 4, 5, 6, 6}
	b := make([]int, 3)
	copy(b, a)
	t.Log(b)
	c := []int{100, 200, 200}
	copy(a, c)
	t.Log(a)
}

func TestTValue_PtrAdd(t *testing.T) {
	s := make([]TValue, 10)
	for i := 0; i < len(s); i++ {
		s[0].PtrAdd(i).SetNumber(LuaNumber(i))
		// t.Log(s[i].NumberValue())
		if !s[i].IsNumber() || s[i].NumberValue() != LuaNumber(i) {
			t.Errorf("PtrAdd() = %v, want %v", s[i].NumberValue(), LuaNumber(i))
		}
	}
}
