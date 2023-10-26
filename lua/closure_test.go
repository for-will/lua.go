package golua

import (
	"testing"
)

func TestClosureHeader_L(t *testing.T) {
	c := &CClosure{
		upValue: make([]TValue, 10),
	}

	for i := range c.upValue {
		c.upValue[i].SetNumber(LuaNumber(i))
	}

	var p Closure = c
	for i, v := range p.C().upValue {
		if v.NumberValue() != LuaNumber(i) {
			t.Errorf("want %v got %v", i, v.NumberValue())
		}
		t.Log(v.NumberValue())
	}
}
