package golua

// 对应C函数：`void luaV_concat (lua_State *L, int total, int last)'
func (L *LuaState) vConcat(total int, last int) {

	for total > 1 { /* repeat until only 1 result left */
		top := L.Base().Ptr(last + 1)
		p1 := top.Ptr(-2)
		p2 := top.Ptr(-1)
		var n = 2 /* number of elements handled in this pass (at least 2) */
		if !(p1.IsString() || p1.IsNumber()) || !toString(L, p2) {
			if !callBinTM(L, p1, p2, p1, TM_CONCAT) {
				L.DebugConcatError(p1, p2)
			}
		} else if p2.StringValue().Len == 0 { /* second op is empty? */
			toString(L, p1) /* result is first op (as string) */
		} else {
			/* at least two string values; get as many as possible */
			tl := top.Ptr(-1).StringValue().Len
			/* collect total length */
			for n = 1; n < total && toString(L, top.Ptr(-n-1)); n++ {
				l := top.Ptr(-n - 1).StringValue().Len
				if l >= int(MAX_SIZET)-tl {
					L.DebugRunError("string length overflow")
				}
				tl += l
			}
			buffer := L.G().buff.OpenSpace(tl)
			tl = 0
			for i := n; i > 0; i-- { /* concat all strings */
				s := top.Ptr(-i).StringValue()
				copy(buffer[tl:], s.Bytes)
				tl += s.Len
			}
			top.Ptr(-n).SetString(L, L.sNewLStr(buffer[:tl]))
		}
		total -= n - 1 /* got 'n' strings to create 1 new */
		last -= n - 1
	}
}

func toString(L *LuaState, obj StkId) bool {
	return obj.IsString() || obj.ToString(L)
}

// 对应C函数：`static int call_binTM (lua_State *L, const TValue *p1, const TValue *p2, StkId res, TMS event)'
func callBinTM(L *LuaState, p1 *TValue, p2 *TValue, res StkId, event TMS) bool {
	// todo: callBinTM
	return false
}
