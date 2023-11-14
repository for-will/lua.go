package golua

import "unsafe"

const MEMERRMSG = "not enough memory"

// 对应C函数：`void *luaM_toobig (lua_State *L)'
func (L *LuaState) mTooBig() interface{} {
	L.gRunError("memory allocation error: block too big")
	return nil /* to avoid warnings */
}

// 对应C函数：`luaM_growvector(L,v,nelems,size,t,limit,e)'
func mGrowVector[T any](L *LuaState, v *[]T, nElems int, size *int, limit int, errMsg string) {
	if nElems+1 > *size {
		*v = mGrowAux_(L, *v, size, limit, errMsg)
	}
}

// 对应C函数：
// `void *luaM_growaux_ (lua_State *L, void *block, int *size, size_t size_elems, int limit, const char *errormsg)‘
func mGrowAux_[T any](L *LuaState, v []T, size *int, limit int, errMsg string) []T {
	const MINSIZEARRAY = 4

	var newSize int
	if *size >= limit/2 { /* cannot double it? */
		if *size >= limit { /* cannot grow even a little? */
			L.gRunError(errMsg)
		}
		newSize = limit /* still have at least one free place */
	} else {
		newSize = *size * 2
		if newSize < MINSIZEARRAY {
			newSize = MINSIZEARRAY /* minimum size */
		}
	}
	var newVector = make([]T, newSize)
	copy(newVector, v)
	*size = newSize /* update only when everything else is OK */
	return newVector
}

// 对应C函数：`luaM_reallocvector(L, v,oldn,n,t)'
func mReallocVector[T any](L *LuaState, v *[]T, oldn int, n int) {
	var e = unsafe.Sizeof((*v)[0])
	*v = mReallocV(L, *v, n, int(e))
}

// 对应C函数：`luaM_reallocv(L,b,on,n,e)'
func mReallocV[T any](L *LuaState, b []T, n int, e int) []T {
	if n+1 <= int(MAX_SIZET)/e { /* +1 to avoid warnings */
		var newV = make([]T, n)
		copy(newV, b)
		return newV
	}

	L.mTooBig()
	return nil
}
