package golua

// FIXEDBIT bit 5 - object is fixed (should not be collected)
const (
	WHITE0BIT = 0
	WHITE1BIT = 1
	BLACKBIT  = 2
	FIXEDBIT  = 5
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
