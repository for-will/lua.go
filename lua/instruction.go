package golua

import (
	"fmt"
	"strconv"
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

func (i *Instruction) DumpCode(getKst func(n int) string) string {

	var (
		REG = func(n int) string {
			return "r" + strconv.Itoa(n)
		}
		KST = func(n int) string {
			return "k" + strconv.Itoa(n) + "::" + getKst(n)
		}
		RA = func() string {
			return REG(i.GetArgA())
		}
		RB = func() string {
			return REG(i.GetArgB())
		}
		RKB = func() string {
			var b = i.GetArgB()
			if ISK(b) {
				return KST(INDEXK(b))
			} else {
				return REG(b)
			}
		}
		RKC = func() string {
			var c = i.GetArgC()
			if ISK(c) {
				return KST(INDEXK(c))
			} else {
				return REG(c)
			}
		}
		KBX = func() string {
			return KST(i.GetArgBx())
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
	case OP_CALL: /* R(A), ... ,R(A+C-2) := R(A)(R(A+1), ... ,R(A+B-1)) */
		var results []string
		var a = i.GetArgA()
		var b = i.GetArgB()
		var c = i.GetArgC()
		for j := a; j <= a+c-2; j++ {
			results = append(results, REG(j))
		}
		var args []string
		for j := a + 1; j <= a+b-1; j++ {
			args = append(args, REG(j))
		}
		if len(results) > 0 {
			desc = fmt.Sprintf("%s := %s(%s)",
				strings.Join(results, ", "), RA(), strings.Join(args, ", "))
		} else {
			desc = fmt.Sprintf("%s(%s)", RA(), strings.Join(args, ", "))
		}
	case OP_RETURN: /* return RA(A), ... ,R(A+B-2) */
		var a = i.GetArgA()
		var b = i.GetArgB()
		var results []string
		for j := a; j < a+b-2; j++ {
			results = append(results, fmt.Sprintf("R(%d)", j))
		}
		desc = "return " + strings.Join(results, ", ")
	case OP_LOADK: /* R(A) := Kst(Bx) */
		desc = fmt.Sprintf("%s := %s", RA(), KBX())
	case OP_SETGLOBAL: /* Gbl[Kst(Bx)] := R(A) */
		desc = fmt.Sprintf("Gbl[%s] := %s", KBX(), RA())
	case OP_MOVE: /* R(A) := R(B) */
		desc = fmt.Sprintf("%s := %s", RA(), RB())
	case OP_SETLIST: /* R(A)[(C-1)*FPF+i] := R(A+i), 1 <= i <= B */
		var a = i.GetArgA()
		var b = i.GetArgB()
		var c = i.GetArgC()

		if b <= 4 {
			var idxs []string
			var vars []string
			for j := 1; j <= b; j++ {
				idxs = append(idxs, strconv.Itoa(LFIELDS_PER_FLUSH*(c-1)+j))
				vars = append(vars, REG(a+j))
			}
			desc = fmt.Sprintf("%s[%s] := %s", RA(),
				strings.Join(idxs, ", "), strings.Join(vars, ", "))
		} else {
			var idxs strings.Builder
			var vars strings.Builder
			idxs.WriteString(strconv.Itoa(LFIELDS_PER_FLUSH*(c-1) + 1))
			idxs.WriteString(", ")
			idxs.WriteString(strconv.Itoa(LFIELDS_PER_FLUSH*(c-1) + 2))
			idxs.WriteString(", ... ")
			idxs.WriteString(strconv.Itoa(LFIELDS_PER_FLUSH*(c-1) + b))

			vars.WriteString(REG(a + 1))
			vars.WriteString(", ")
			vars.WriteString(REG(a + 2))
			vars.WriteString(", ... ")
			vars.WriteString(REG(a + b))

			desc = fmt.Sprintf("%s[%s] := %s", RA(), idxs.String(), vars.String())
		}

	default:
		desc = "..."
	}
	return opInfo + desc
}
