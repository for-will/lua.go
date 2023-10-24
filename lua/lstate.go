package golua

// GCObject Union of all collectable objects
//type GCObject interface{}

type GCObject interface {
	Next() GCObject
	SetNext(obj GCObject)
	ToString() *TString // gco2ts
}

type GlobalState struct {
	StrT *StringTable // hash table for strings
}

func (g *GlobalState) LuaCWhite() lu_byte {
	//todo:
	return 0
}

type LuaState struct {
	CommonHeader
	lG *GlobalState
}

func (L *LuaState) G() *GlobalState {
	return L.lG
}
