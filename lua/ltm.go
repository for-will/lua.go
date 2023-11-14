package golua

type TMS = int /* 对就C类型：enum TMS*/

const (
	TM_INDEX TMS = iota
	TM_NEWINDEX
	TM_GC
	TM_MODE
	TM_EQ /* last tag method with `fast' access */
	TM_ADD
	TM_SUB
	TM_MUL
	TM_DIV
	TM_MOD
	TM_POW
	TM_UNM
	TM_LEN
	TM_LT
	TM_LE
	TM_CONCAT
	TM_CALL
	TM_N /* number of elements in the enum */
)

// 对应C函数：`gfasttm(g,et,e)'
func gFastTM(g *GlobalState, et *Table, e TMS) *TValue {
	if et == nil {
		return nil
	}
	if et.flags&(1<<lu_byte(e)) != 0 {
		return nil
	}
	return tGetTM(et, e, g.tmName[e])
}

// FastTM
// 对应C函数：`fasttm(l,et,e)'
func FastTM(L *LuaState, mt *Table, e TMS) *TValue {
	return gFastTM(L.G(), mt, e)
}

// function to be used with macro "fasttm": optimized for absence of
// tag methods.
// 对应C函数：`const TValue *luaT_gettm (Table *events, TMS event, TString *ename)'
func tGetTM(events *Table, event TMS, ename *TString) *TValue {
	tm := events.GetByString(ename)
	LuaAssert(event <= TM_EQ)
	if tm.IsNil() { /* no tag method? */
		events.flags |= 1 << event /* cache this fact */
		return nil
	}
	return tm
}

// 对应C函数：`const TValue *luaT_gettmbyobj (lua_State *L, const TValue *o, TMS event)'
func (L *LuaState) tGetTMByObj(o *TValue, event TMS) *TValue {
	var mt *Table
	switch o.gcType() {
	case LUA_TTABLE:
		mt = o.TableValue().metatable
	case LUA_TUSERDATA:
		mt = o.UdataValue().metatable
	default:
		mt = L.G().mt[o.gcType()]
	}
	if mt != nil {
		return mt.GetByString(L.G().tmName[event])
	} else {
		return LuaObjNil
	}
}

// 对应C函数：`void luaT_init (lua_State *L)'
func (L *LuaState) tInit() {
	var luaTEventName = []string{ /* ORDER TM*/
		"__index", "__newindex",
		"__gc", "__mode", "__eq",
		"__add", "__sub", "__mul", "__div", "__mod",
		"__pow", "__unm", "__len", "__lt", "__le",
		"__concat", "__call",
	}
	for i := 0; i < TM_N; i++ {
		L.G().tmName[i] = L.sNew([]byte(luaTEventName[i]))
		L.G().tmName[i].Fix() /* never collect these names */
	}
}

var LuaTTypeNames = [...]string{
	"nil", "boolean", "userdata", "number",
	"string", "table", "function", "userdata", "thread",
	"proto", "upval",
}
