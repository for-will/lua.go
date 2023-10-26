package golua

import "testing"

func TestLuaState_parser(t *testing.T) {
	if LUA_SIGNATURE[0] != byte(27) {
		t.Error("invalid LUA_SIGNATURE: ", []byte(LUA_SIGNATURE))
	}
	t.Log([]byte(LUA_SIGNATURE))
	t.Log(LUA_SIGNATURE)
	var name []byte = nil

	t.Log(len(name))
}

func TestLuaState_dRawRunProtected(t *testing.T) {
	L := &LuaState{}
	status := L.dRawRunProtected(func(L *LuaState, ud interface{}) {
		L.errorJmp.status = 100
		panic("panic in func")
		L.errorJmp.status = 200
	}, nil)

	if status != 100 {
		t.Errorf("want 100 got %v", status)
	}
}
