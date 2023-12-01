package mem

import (
	"math"
	"unsafe"
)

const (
	MAX_SIZET = (^uint32(0)) - 2
	MAX_INT   = math.MaxInt32 - 2

	MEMERRMSG = "not enough memory"
)

type Vec[T any] []T

type ErrorHandler interface {
	DbgRunError(format string, args ...interface{})
}

func (v *Vec[T]) Size() int {
	return len(*v)
}

// ErrTooBig
// 对应C函数：`void *luaM_toobig (lua_State *L)'
func ErrTooBig(h ErrorHandler) {
	h.DbgRunError("memory allocation error: block too big")
}

// ReAlloc
// 对应C函数：`luaM_reallocvector(L, v,oldn,n,t)'
func (v *Vec[T]) ReAlloc(n int, h ErrorHandler) {
	if uintptr(n+1) <= uintptr(MAX_SIZET)/unsafe.Sizeof(*(*T)(nil)) { /* +1 to avoid warnings */
		var v2 = make([]T, n)
		copy(v2, *v)
		*v = v2
	} else {
		ErrTooBig(h)
	}
}

// Free
// 对应C函数：`luaM_free(L, b)'
func (v *Vec[T]) Free(h ErrorHandler) {
	v.ReAlloc(0, h)
}

// Init 初始化，创建分配大小为size的T的切片。
// 对应C函数：`luaM_newvector(L,n,t)'
func (v *Vec[T]) Init(size int, h ErrorHandler) {
	/* 这里不使用ReAlloc(size, h)，因为ReAlloc中会进行copy，在这里不需要copy旧的值到*v中 */
	if uintptr(size+1) <= uintptr(MAX_SIZET)/unsafe.Sizeof(*(*T)(nil)) { /* +1 to avoid warnings */
		*v = make([]T, size)
	} else {
		ErrTooBig(h)
	}
}

// Grow
// 对应C函数：`luaM_growvector(L,v,nelems,size,t,limit,e)'
func (v *Vec[T]) Grow(h ErrorHandler, n int, limit int, errMsg string) {
	if n+1 > v.Size() {
		v.growAux_(h, limit, errMsg)
	}
}

// 对应C函数：
// `void *luaM_growaux_ (lua_State *L, void *block, int *size, size_t size_elems, int limit, const char *errormsg)‘
func (v *Vec[T]) growAux_(h ErrorHandler, limit int, errMsg string) {
	const MINSIZEARRAY = 4

	var size = len(*v)
	var sz int
	if size >= limit/2 { /* cannot double it? */
		if size >= limit { /* cannot grow even a little? */
			h.DbgRunError(errMsg)
		}
		sz = limit /* still have at least one free place */
	} else {
		sz = size * 2
		if sz < MINSIZEARRAY {
			sz = MINSIZEARRAY /* minimum size */
		}
	}
	var v2 = make([]T, sz)
	copy(v2, *v)
	*v = v2
}

// ElemIndex 返回e在arr中的位置
func ElemIndex[T any](arr []T, e *T) int {
	var offset = uintptr(unsafe.Pointer(e)) - uintptr(unsafe.Pointer(&arr[0]))
	return int(offset / unsafe.Sizeof(arr[0]))
}
