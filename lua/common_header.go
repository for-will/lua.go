package golua

import "C"
import "unsafe"

type CommonHeader struct {
	next GCObject
	tt
	marked lu_byte
}

func (c *CommonHeader) Next() GCObject {
	return c.next
}

func (c *CommonHeader) SetNext(obj GCObject) {
	c.next = obj
}

func (c *CommonHeader) ToString() *TString {
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
