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

func (i *Instruction) DumpCode(getKst func(n int) string, top int) string {

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
		Regs = func(a, b int) string {
			var cnt = b - a + 1
			if cnt <= 4 {
				var regs []string
				for j := a; j <= b; j++ {
					regs = append(regs, "r"+strconv.Itoa(j))
				}
				return strings.Join(regs, ", ")
			} else {
				var s strings.Builder
				s.WriteString("r" + strconv.Itoa(a) + ", ")
				s.WriteString("r" + strconv.Itoa(a+1) + ", ")
				s.WriteString("... ")
				s.WriteString("r" + strconv.Itoa(b))
				return s.String()
			}
		}
		Idxs = func(a, b int) string {
			var cnt = b - a + 1
			if cnt <= 4 {
				var regs []string
				for j := a; j <= b; j++ {
					regs = append(regs, strconv.Itoa(j))
				}
				return strings.Join(regs, ", ")
			} else {
				var s strings.Builder
				s.WriteString(strconv.Itoa(a) + ", ")
				s.WriteString(strconv.Itoa(a+1) + ", ")
				s.WriteString("... ")
				s.WriteString(strconv.Itoa(b))
				return s.String()
			}
		}
		BOOL = func(v int) string {
			return []string{"false", "true"}[v]
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
		var a = i.GetArgA()
		var b = i.GetArgB()
		var c = i.GetArgC()

		var results, args string
		if c == 0 {
			results = RA() + "..."
		} else {
			results = Regs(a, a+c-2)
		}
		if b == 0 {
			args = Regs(a+1, top-1)
		} else {
			args = Regs(a+1, a+b-1)
		}
		if len(results) > 0 {
			desc = fmt.Sprintf("%s := %s(%s)", results, RA(), args)
		} else {
			desc = fmt.Sprintf("%s(%s)", RA(), args)
		}
	case OP_RETURN: /* return RA(A), ... ,R(A+B-2) */
		var a = i.GetArgA()
		var b = i.GetArgB()
		desc = "return " + Regs(a, a+b-2)
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
		var idxs = Idxs(LFIELDS_PER_FLUSH*(c-1)+1, LFIELDS_PER_FLUSH*(c-1)+b)
		var vars = Regs(a+1, a+b)
		desc = fmt.Sprintf("%s[%s] := %s", RA(), idxs, vars)
	case OP_CLOSURE: /* R(A) := closure(KPROTO[Bx], R(A), ... .R(A+n)) */
		desc = fmt.Sprintf("%s := closure(kproto[%d])", RA(), i.GetArgBx())
	case OP_ADD: /* R(A) := RK(B) + RK(C) */
		desc = fmt.Sprintf("%s := %s + %s", RA(), RKB(), RKC())
	case OP_MUL: /* R(A) := RK(B) * RK(C) */
		desc = fmt.Sprintf("%s := %s * %s", RA(), RKB(), RKC())
	case OP_GETUPVAL: /* R(A) := UpValue[B] */
		desc = fmt.Sprintf("%s := upvalue[%d]", RA(), i.GetArgB())
	case OP_SETUPVAL: /* UpValue[B] := R(A) */
		desc = fmt.Sprintf("upvalue[%d] := %s", i.GetArgB(), RA())
	case OP_SELF: /* R(A+1) := R(B); R(A) := R(B)[RK(C)] */
		desc = fmt.Sprintf("%s := %s[%s]; %s\u001B[33m<self>\u001B[34m := %s",
			RA(), RB(), RKC(), REG(i.GetArgA()+1), RB())
	case OP_CONCAT: /* R(A) := R(B).. ... ..R(C) */
		desc = fmt.Sprintf("%s := concat(%s)", RA(), Regs(i.GetArgB(), i.GetArgC()))
	case OP_EQ: /* if ((RK(B) == RK(C)) ~= A) then pc++ */
		if i.GetArgA() == 0 {
			desc = fmt.Sprintf("if (%s == %s) then pc++", RKB(), RKC())
		} else {
			desc = fmt.Sprintf("if (%s != %s) then pc++", RKB(), RKC())
		}
	case OP_LOADBOOL: /* R(A) := (Bool)B; if (C) pc++ */
		desc = fmt.Sprintf("%s := %s; if (%s) pc++",
			RA(), BOOL(i.GetArgB()), BOOL(i.GetArgC()))
	case OP_JMP: /* pc+=sBx */
		desc = fmt.Sprintf("pc += %d", i.GetArgSBx())
	case OP_TEST: /* if not (R(A) <=> C) then pc++ */
		desc = fmt.Sprintf("if (%s is %s) then pc++", RA(), BOOL((i.GetArgC()+1)%2))
	case OP_TESTSET: /* if (R(B) <=> C) then R(A) := R(B) else pc++ */
		desc = fmt.Sprintf("if (%s is %s) then %s := %s else pc++",
			RB(), BOOL(i.GetArgC()), RA(), RB())
	case OP_FORPREP: /* R(A)-=R(A+2); pc+=sBx */
		desc = fmt.Sprintf("%s -= %s; pc += %d", RA(), REG(i.GetArgA()+2), i.GetArgSBx())
	case OP_FORLOOP: /* R(A)+=R(A+2); if R(A) <?= R(A+1) then { pc+=sBx; R(A+3)=R(A) }  */
		var a = i.GetArgA()
		desc = fmt.Sprintf("%s+=%s; if %s <?= %s then { pc+=%d; %s=%s }",
			RA(), REG(a+2), RA(), REG(a+2), i.GetArgSBx(), REG(a+3), RA())
	case OP_TFORLOOP: /* R(A+3), ... ,R(A+2+C) := R(A)(R(A+1), R(A+2)) */
		/* if R(A+3) ~= nil then R(A+2)=R(A+3) else pc++ */
		var a = i.GetArgA()
		var c = i.GetArgC()
		desc = fmt.Sprintf("%s := %s(%s); \n", Regs(a+3, a+2+c), RA(), Regs(a+1, a+2))
		desc += fmt.Sprintf("%31s// if %s ~= nil then %s=%s else pc++",
			" ", REG(a+3), REG(a+2), REG(a+3))

	default:
		desc = "..."
	}
	return opInfo + desc
}
