package golua

import (
	"reflect"
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
		s[0].Ptr(i).SetNumber(LuaNumber(i))
		// t.Log(s[i].NumberValue())
		if !s[i].IsNumber() || s[i].NumberValue() != LuaNumber(i) {
			t.Errorf("PtrAdd() = %v, want %v", s[i].NumberValue(), LuaNumber(i))
		}
	}
}

func Test_oPushVfString1(t *testing.T) {
	type args struct {
		L      *LuaState
		format []byte
		argv   []interface{}
	}
	tests := []struct {
		name string
		args args
		want []byte
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.args.L.oPushVfString(tt.args.format, tt.args.argv...); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("oPushVfString() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_oStr2d(t *testing.T) {
	var num LuaNumber
	t.Log(oStr2d("123  \r   \n", &num))
}
