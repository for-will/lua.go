package golua

import (
	"luar/lua/mem"
	"unsafe"
)

// Proto Function Prototypes
// 对应C结构体：`struct Proto`
type Proto struct {
	CommonHeader
	k               mem.Vec[TValue]      /* constants used by the function */
	code            mem.Vec[Instruction] /* */
	p               mem.Vec[*Proto]      /* functions defined inside the function */
	lineInfo        mem.Vec[int]         /* map from opcodes to source lines */
	locVars         mem.Vec[LocVar]      //
	upValues        mem.Vec[*TString]    /* upvalue names*/
	source          *TString             /* */
	lineDefined     int                  /* */
	lastLineDefined int                  /* */
	gcList          *GCObject            /* */
	nUps            int                  /* number of up-values */
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
