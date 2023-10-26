package golua

// LocVar
// 对应C结构体：`struct LocVar`
type LocVar struct {
	varName *TString
	startPc int /* first point where variable is active */
	endPc   int /* first point where variable is dead */
}
