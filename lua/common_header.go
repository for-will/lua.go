package golua

import "C"
import "unsafe"

type CommonHeader struct {
	next GCObject
	tt
	marked lu_byte
}

func (c *CommonHeader) GetNext() GCObject {
	return c.next
}

func (c *CommonHeader) SetNext(obj GCObject) {
	c.next = obj
}

func (c *CommonHeader) SetMarked(m lu_byte) {
	c.marked = m
}

func (c *CommonHeader) setType(t ttype) {
	c.tt = t
}
func (c *CommonHeader) ToTString() *TString {
	LuaAssert(c.IsString())
	return (*TString)(unsafe.Pointer(c))
}

func (c *CommonHeader) ToTable() *Table {
	LuaAssert(c.IsTable())
	return (*Table)(unsafe.Pointer(c))
}

func (c *CommonHeader) ToClosure() Closure {
	LuaAssert(c.IsFunction())
	return (*ClosureHeader)(unsafe.Pointer(c))
}

func (c *CommonHeader) ToUpval() *UpVal {
	LuaAssert(c.IsUpval())
	return (*UpVal)(unsafe.Pointer(c))
}

func (c *CommonHeader) ToUdata() *Udata {
	LuaAssert(c.IsUserdata())
	return (*Udata)(unsafe.Pointer(c))
}

func (c *CommonHeader) ToThread() *LuaState {
	LuaAssert(c.IsThread())
	return (*LuaState)(unsafe.Pointer(c))
}
