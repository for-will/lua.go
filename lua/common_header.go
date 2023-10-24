package golua

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
	//TODO implement me
	panic("implement me")
}
