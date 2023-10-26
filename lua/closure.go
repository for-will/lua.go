package golua

import "unsafe"

// LuaCFunction
// 对应C类型：`typedef int (*lua_CFunction) (lua_State *L)`
type LuaCFunction func(L *LuaState)

type ClosureHeader struct {
	CommonHeader
	isC       bool
	nUpValues lu_byte
	gcList    *GCObject
	env       *Table
}

// CClosure
// 对应C结构体：`struct CClosure`
type CClosure struct {
	ClosureHeader
	f       LuaCFunction
	upValue []TValue
}

// LClosure
// 对应C结构体：`struct LClosure`
type LClosure struct {
	ClosureHeader
	p      *Proto
	upVals []*UpVal
}

// Closure
// 对应C结构体：`union Closure`
type Closure interface {
	IsCFunction() bool
	IsLFunction() bool
	C() *CClosure
	L() *LClosure
}

func (ch *ClosureHeader) IsCFunction() bool {
	return ch.IsFunction() && ch.isC
}

func (ch *ClosureHeader) IsLFunction() bool {
	return ch.IsFunction() && !ch.isC
}

func (ch *ClosureHeader) C() *CClosure {
	return (*CClosure)(unsafe.Pointer(ch))
}

func (ch *ClosureHeader) L() *LClosure {
	return (*LClosure)(unsafe.Pointer(ch))
}

// func (lc *LClosure) ToClosure() Closure {
// 	return lc
// }
//
// func (cc *CClosure) ToClosure() Closure {
// 	return cc
// }
