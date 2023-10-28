package golua

import "log"

// FIXEDBIT bit 5 - object is fixed (should not be collected)
const (
	WHITE0BIT = 0
	WHITE1BIT = 1
	BLACKBIT  = 2
	FIXEDBIT  = 5
	WHITEBITS = 1<<WHITE0BIT | 1<<WHITE1BIT
)

// IsWhite
// 对应C函数：`iswhite(x)'
func (c *CommonHeader) IsWhite() bool {
	return (c.marked & (1<<WHITE0BIT | 1<<WHITE1BIT)) != 0
}

// IsBlack
// 对应C函数：`isblack(x)'
func (c *CommonHeader) IsBlack() bool {
	return (c.marked & (1 << BLACKBIT)) != 0
}

// IsGray
// 对应C函数：`isgray(x)'
func (c *CommonHeader) IsGray() bool {
	return !c.IsBlack() && !c.IsWhite()
}

// 对应C函数：`isdead(g,v)'
func isdead(g *GlobalState, v GCObject) bool {
	// todo: isdead
	log.Println("isdead: not implemented ")
	return false
}

// 对应C函数：`luaC_checkGC(L)'
func (L *LuaState) cCheckGC() {
	// todo: cCheckGC
	log.Println("cCheckGC not implemented")
}
