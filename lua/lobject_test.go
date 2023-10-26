package golua

import (
	"fmt"
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

func Test_oPushVfString(t *testing.T) {
	outputArgs := func(args ...interface{}) {
		// a := args[0].([]byte)
		switch args[0].(type) {
		case nil:
			t.Log("0 is nil")
		case []byte:
			t.Log("0 is []byte ", args[0].([]byte))
		case string:
			t.Log("0 is string ", args[0].(string))
		default:
			t.Log("what's 0")
		}

		// t.Log("0 is ", args[0].([]byte))
		// b := args[1].(int32)
		// t.Log("b = ", b)
		t.Log(fmt.Sprintf("%p", args[0]))
	}

	outputArgs(t, 'a')
}
