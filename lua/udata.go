package golua

type Udata struct {
	CommonHeader
	metatable *Table
	env       *Table
	len       int
}
