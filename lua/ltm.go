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

// 对应C函数：`const TValue *luaT_gettmbyobj (lua_State *L, const TValue *o, TMS event)'
func (L *LuaState) tGetTMByObj(o *TValue, event TMS) *TValue {
	var mt *Table
	switch o.Type() {
	case LUA_TTABLE:
		mt = o.TableValue().metatable
	case LUA_TUSERDATA:
		mt = o.UdataValue().metatable
	default:
		mt = L.G().mt[o.Type()]
	}
	if mt != nil {
		return mt.GetByString(L.G().tmname[event])
	} else {
		return LuaObjNil
	}
}
