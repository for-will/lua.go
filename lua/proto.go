package golua

import "unsafe"

// Proto Function Prototypes
// 对应C结构体：`struct Proto`
type Proto struct {
	CommonHeader
	k               []TValue      /* constants used by the function */
	code            []Instruction /* */
	p               []*Proto      /* functions defined inside the function */
	lineInfo        []int         /* map from opcodes to source lines */
	locVars         []LocVar      //
	upValues        []*TString    /* upvalue names*/
	source          *TString      /* */
	sizeUpValues    int           /* */
	sizeK           int           /* size of `k` */
	sizeCode        int           /* */
	sizeLineInfo    int           /* */
	sizeP           int           /* size of `p` */
	sizeLocVars     int           /* */
	lineDefined     int           /* */
	lastLineDefined int           /* */
	gcList          *GCObject     /* */
	nUps            int           /* number of up-values */
	numParams       int
	isVarArg        lu_byte
	maxStackSize    int
}

// 对应C函数：`pcRel(pc, p)'
func (p *Proto) pcRel(pc *Instruction) int {
	n := uintptr(unsafe.Pointer(pc)) - uintptr(unsafe.Pointer(&p.code[0]))
	return int(n/unsafe.Sizeof(Instruction(0))) - 1 // 为什么要-1？
}

// 对应C函数：`getline(f,pc)'
func (p *Proto) getLine(pc int) int {
	if p.lineInfo != nil {
		return p.lineInfo[pc]
	}
	return 0
}
