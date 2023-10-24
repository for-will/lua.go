package golua

import "math"

type lu_byte = uint8

//#define MAX_SIZET	((size_t)(~(size_t)0)-2)

const MAX_SIZET = (^uint32(0)) - 2

const MAX_INT = math.MaxInt32 - 2

func LuaAssert(c bool) {
	// todo
}
func CheckExp(c bool) {
	// todo
	if !c {
		panic("CheckExp Failed")
	}
}
