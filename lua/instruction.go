package golua

import (
	"fmt"
	"strings"
	"unsafe"
)

// Instruction 对应C类型`Instruction`
// type for virtual-machine instructions
// must be an unsigned with (at least) 4 bytes (see details in lopcodes.h)
type Instruction uint32

func (i *Instruction) Ptr(n int) *Instruction {
	if n >= 0 {
		p := uintptr(unsafe.Pointer(i)) + uintptr(n)*unsafe.Sizeof(Instruction(0))
		return (*Instruction)(unsafe.Pointer(p))
	} else {
		p := uintptr(unsafe.Pointer(i)) - uintptr(-n)*unsafe.Sizeof(Instruction(0))
		return (*Instruction)(unsafe.Pointer(p))
	}
}

func (i *Instruction) String() string {

	var (
		RA = func() string {
			return fmt.Sprintf("R(%d)", i.GetArgA())
		}
		RB = func() string {
			return fmt.Sprintf("R(%d)", i.GetArgB())
		}
		RKB = func() string {
			var b = i.GetArgB()
			if ISK(b) {
				return fmt.Sprintf("K(%d)", INDEXK(b))
			} else {
				return fmt.Sprintf("R(%d)", b)
			}
		}
		RKC = func() string {
			var c = i.GetArgC()
			if ISK(c) {
				return fmt.Sprintf("K(%d)", INDEXK(c))
			} else {
				return fmt.Sprintf("R(%d)", c)
			}
		}
		KBX = func() string {
			return fmt.Sprintf("K(%d)", i.GetArgBx())
		}
	)
	var op = i.GetOpCode()
	var opName = op.String()
	var opInfo string
	switch op.getOpMode() {
	case iABC:
		var a = i.GetArgA()
		var b = i.GetArgB()
		var c = i.GetArgC()
		opInfo = fmt.Sprintf("%-10s %d, %d, %d", opName, a, b, c)
	case iABx:
		var a = i.GetArgA()
		var bx = i.GetArgBx()
		opInfo = fmt.Sprintf("%-10s %d, %d", opName, a, bx)
	case iAsBx:
		var a = i.GetArgA()
		var bx = i.GetArgSBx()
		opInfo = fmt.Sprintf("%-10s %d, %d", opName, a, bx)
	default:
		panic("unknown op-mode")
	}
	opInfo = fmt.Sprintf("%-30s // ", opInfo)
	// var a = i.GetArgA()
	var desc string
	switch op {
	case OP_NEWTABLE:
		var b = i.GetArgB()
		var c = i.GetArgC()
		desc = fmt.Sprintf("%s := NewTable(nArray: %d, nHash: %d)", RA(), oFb2Int(b), oFb2Int(c))
	case OP_SETTABLE: // R(A)[RK(B)] := RK(C)
		desc = fmt.Sprintf("%s[%s] := %s", RA(), RKB(), RKC())
	case OP_GETGLOBAL: // R(A) := Gbl[Kst(Bx)]
		desc = fmt.Sprintf("%s := Gbl[%s]", RA(), KBX())
	case OP_GETTABLE: // R(A) := R(B)[RK(C)]
		desc = fmt.Sprintf("%s := %s[%s]", RA(), RB(), RKC())
	case OP_CALL: // R(A), ... ,R(A+C-2) := R(A)(R(A+1), ... ,R(A+B-1))
		var results []string
		var a = i.GetArgA()
		var b = i.GetArgB()
		var c = i.GetArgC()
		for j := a; j <= a+c-2; j++ {
			results = append(results, fmt.Sprintf("R(%d)", j))
		}
		var args []string
		for j := a + 1; j <= a+b-1; j++ {
			args = append(args, fmt.Sprintf("R(%d)", j))
		}
		if len(results) > 0 {
			desc = fmt.Sprintf("%s := %s(%s)",
				strings.Join(results, ", "), RA(), strings.Join(args, ", "))
		} else {
			desc = fmt.Sprintf("%s(%s)", RA(), strings.Join(args, ", "))
		}
	case OP_RETURN: // return RA(A), ... ,R(A+B-2)
		var a = i.GetArgA()
		var b = i.GetArgB()
		var results []string
		for j := a; j < a+b-2; j++ {
			results = append(results, fmt.Sprintf("R(%d)", j))
		}
		desc = "return " + strings.Join(results, ", ")
	default:
		desc = "..."
	}
	return opInfo + desc
}
