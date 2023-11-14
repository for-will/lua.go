package golua

/*===========================================================================
  We assume that instructions are unsigned numbers.
  All instructions have an opcode in the first 6 bits.
  Instructions cae have the following fields:
  `A' : 8 bits
  `B' : 9 bits
  `C' : 9 bits
  `Bx' : 18 bits (`B' and `C' together)
  `sBx' : signed Bx

  A signed argument is represented in excess K; that is, the number
  value is the unsigned value minus K. K is exactly the maximum value
  for that argument (so that -max is represented by 0, and +max is
  represented by 2*max), which is half the maximum for the corresponding
  unsigned argument.
===========================================================================*/

type OpMode int

/* basic instruction format */
const (
	iABC OpMode = iota
	iABx
	iAsBx
)

/* size and positon of opcode arguments */
const (
	SIZE_C  = 9
	SIZE_B  = 9
	SIZE_Bx = SIZE_C + SIZE_B
	SIZE_A  = 8

	SIZE_OP = 6

	POS_OP = 0
	POS_A  = POS_OP + SIZE_OP
	POS_C  = POS_A + SIZE_A
	POS_B  = POS_C + SIZE_C
	POS_Bx = POS_C
)

const (
	MAXARG_Bx  = 1<<SIZE_Bx - 1
	MAXARG_sBx = MAXARG_Bx >> 1 /* `sBx' is signed */

	MAXARG_A = 1<<SIZE_A - 1
	MAXARG_B = 1<<SIZE_B - 1
	MAXARG_C = 1<<SIZE_C - 1
)

// MASK1
// create a mask with `n' 1 bits at position `p'
// 对应C函数：`MASK1(n,p)'
func MASK1(n int, p int) Instruction {
	return (^(^Instruction(0) << n)) << p
}

// MASK0
// creates a mask with `n' 0 bits at position `p'
// 对应C函数：`MASK0(n,p)
func MASK0(n int, p int) Instruction {
	return ^MASK1(n, p)
}

// GetOpCode
// 对应C函数：`GET_OPCODE(i)'
func (i *Instruction) GetOpCode() OpCode {
	return OpCode(*i >> POS_OP & MASK1(SIZE_OP, 0))
}

// SetOpCode
// 对应C函数：`SET_OPCODE(i,o)'
func (i *Instruction) SetOpCode(op OpCode) {
	*i = (*i & MASK0(SIZE_OP, POS_OP)) |
		((Instruction(op) << POS_OP) & MASK1(SIZE_OP, POS_OP))
}

// GetArgA
// 对应C函数：`GETARG_A(i)'
func (i *Instruction) GetArgA() int {
	return int(*i >> POS_A & MASK1(SIZE_A, 0))
}

// SetArgA
// 对应C函数：`SETARG_A(i,u)'
func (i *Instruction) SetArgA(a int) {
	*i = (*i & MASK0(SIZE_A, POS_A)) |
		((Instruction(a) << POS_A) & MASK1(SIZE_A, POS_A))
}

// GetArgB
// 对应C函数：`GETARG_B(i)'
func (i *Instruction) GetArgB() int {
	return int(*i >> POS_B & MASK1(SIZE_B, 0))
}

// SetArgB
// 对应C函数：`SETARG_B(i,b)'
func (i *Instruction) SetArgB(b int) {
	*i = (*i & MASK0(SIZE_B, POS_B)) |
		((Instruction(b) << POS_Bx) & MASK1(SIZE_B, POS_B))
}

// GetArgC
// 对应C函数：`GETARG_C(i)'
func (i *Instruction) GetArgC() int {
	return int(*i >> POS_C & MASK1(SIZE_B, 0))
}

// SetArgC
// 对应C函数：`SETARG_C(i,b)'
func (i *Instruction) SetArgC(c int) {
	*i = (*i & MASK0(SIZE_C, POS_C)) |
		((Instruction(c) << POS_C) & MASK1(SIZE_C, POS_C))
}

// GetArgBx
// 对应C函数：`GETARG_Bx(i)'
func (i *Instruction) GetArgBx() int {
	return int(*i >> POS_Bx & MASK1(SIZE_Bx, 0))
}

// SetArgBx
// 对应C函数：`SETARG_Bx(i,b)'
func (i *Instruction) SetArgBx(b int) {
	*i = (*i & MASK0(SIZE_Bx, POS_Bx)) |
		((Instruction(b) << POS_Bx) & MASK1(SIZE_Bx, POS_Bx))
}

// GetArgSBx
// 对应C函数：`GETARG_sBx(i)'
func (i *Instruction) GetArgSBx() int {
	return i.GetArgBx() - MAXARG_sBx
}

// SetArgSBx
// 对应C函数：`SETARG_sBx(i,b)'
func (i *Instruction) SetArgSBx(b int) {
	i.SetArgBx(b + MAXARG_sBx)
}

// CreateABC
// 对应C函数：`CREATE_ABC(o,a,b,c)'
func CreateABC(o OpCode, a int, b int, c int) Instruction {
	return Instruction(o)<<POS_OP | Instruction(a)<<POS_A |
		Instruction(b)<<POS_B | Instruction(c)<<POS_C
}

// CreateABx
// 对应C函数：`CREATE_ABx(o,a,bc)'
func CreateABx(o OpCode, a int, bc int) Instruction {
	return Instruction(o)<<POS_OP | Instruction(a)<<POS_A | Instruction(bc)<<POS_Bx
}

/* Macros to operate RK indices */
const (
	BITRK      = 1 << (SIZE_B - 1) /* 最高位置位表示是常量 */
	MAXINDEXRK = BITRK - 1         /* 最大的常量index */
)

// ISK
// test whether value is a constant
// 对应C函数：`ISK(x)'
func ISK(x int) bool {
	return x&BITRK != 0
}

// INDEXK 通过复位最高位的常量标志位，得到低位表示的常量index。
// gets the index of the constant
// 对应C函数：`INDEXK(r)'
func INDEXK(r int) int {
	return r & ^BITRK
}

// RKASK 通过置位常量标志位，返回一个表示常量index的值。
// code a constant index as a RK value
// 对应C函数：`RKASK(x)'
func RKASK(x int) int {
	return x | BITRK
}

const NO_REG = MAXARG_A /* invalid register that fits in 8 bits */

type OpCode int

// R(x) - register
// Kst(x) - constant (in constant table)
// RK(x) == if ISK(x) then Kst(INDEXK(x)) else R(x)
/* grep "ORDER OP" if you change these enums */
const (
	OP_MOVE     OpCode = iota /* A B    R(A) := R(B)                 */
	OP_LOADK                  /* A Bx   R(A) := Kst(Bx)              */
	OP_LOADBOOL               /* A B C  R(A) := (Bool)B; if (C) pc++ */
	OP_LOADNIL                /* A B    R(A) := ... := R(B) := nil   */
	OP_GETUPVAL               /* A B    R(A) := UpValue[B]          */

	OP_GETGLOBAL /* A Bx    R(A) := Gbl[Kst(Bx)]    */
	OP_GETTABLE  /* A B C   R(A) := R(B)[RK(C)]     */

	OP_SETGLOBAL /* A Bx    Gbl[Kst(Bx)] := R(A)    */
	OP_SETUPVAL  /* A B     UpValue[B] := R(A)      */
	OP_SETTABLE  /* A B C   R(A)[RK(B)] := RK(C)    */

	OP_NEWTABLE /* A B C    R(A) := {} (size = B, c)    */

	OP_SELF /* A B C    R(A+1) := R(B); R(A) := R(B)[RK(C)] */

	OP_ADD /* A B C R(A) := RK(B) + RK(C)       */
	OP_SUB /* A B C R(A) := RK(B) - RK(C)       */
	OP_MUL /* A B C R(A) := RK(B) * RK(C)       */
	OP_DIV /* A B C R(A) := RK(B) / RK(C)       */
	OP_MOD /* A B C R(A) := RK(B) % RK(C)       */
	OP_POW /* A B C R(A) := RK(B) ^ RK(C)       */
	OP_UNM /* A B   R(A) := -R(B)               */
	OP_NOT /* A B   R(A) := not R(B)            */
	OP_LEN /* A B   R(A) := length of R(B)      */

	OP_CONCAT /* A B C  R(A) := R(B).. ... ..R(C)   */

	OP_JMP /* sBx       pc+=sBx     */

	OP_EQ /* A B C if ((RK(B) == RK(C)) ~= A) then pc++ */
	OP_LT /* A B C if ((RK(B) <  RK(C)) ~= A) then pc++ */
	OP_LE /* A B C if ((RK(C) <= RK(C)) ~= A) then pc++ */

	OP_TEST    /* A C   if not (R(A) <=> C) then pc++   */
	OP_TESTSET /* A B C     if (R(B) <=> C) then R(A) := R(B) else pc++ */

	OP_CALL     /* A B C    R(A), ... ,R(A+C-2) := R(A)(R(A+1), ... ,R(A+B-1))  */
	OP_TAILCALL /* A B C    return R(A)(R(A+1), ... ,R(A+B-1)                   */
	OP_RETURN   /* A B      return RA(A), ... ,R(A+B-2) (see note)              */

	OP_FORLOOP /* A sBx     R(A)+=R(A+2); if R(A) <?= R(A+1) then { pc+=sBx; R(A+3)=R(A) }            */
	OP_FORPREP /* A sBx     R(A)-=R(A+2); pc+=sBx   */

	OP_TFORLOOP /* A C      R(A+3), ... ,R(A+2+C) := R(A)(R(A+1), R(A+2));
	*                           if R(A+3) ~= nil then R(A+2)=R(A+3) else pc++ */
	OP_SETLIST /* A B C     R(A)[(C-1)*FPF+i] := R(A+i), 1 <= i <= B    */

	OP_CLOSE   /* A         close all variables in the stack upto (>=) R(A) */
	OP_CLOSURE /* A Bx      R(A) := closure(KPROTO[Bx], R(A), ... .R(A+n))  */

	OP_VARARG /* A B        R(A+1), ..., R(A+B-1) = vararg  */
)

const NUM_OPCODES = OP_VARARG + 1

type OpArgMask = lu_byte

const (
	OpArgN = iota /* argument is not used */
	OpArgU        /* argument is used */
	OpArgR        /* argument is a register or a jump offset */
	OpArgK        /* argument is a constant or register/constant */
)

func getOpMode(m OpCode) OpMode {
	return OpMode(luaP_opmodes[m] & 3)
}

func getBMode(m OpCode) OpArgMask {
	return OpArgMask((luaP_opmodes[m] >> 4) & 3)
}

func getCMode(m OpCode) OpArgMask {
	return OpArgMask((luaP_opmodes[m] >> 2) & 3)
}

func testAMode(m OpCode) bool {
	return luaP_opmodes[m]&(1<<6) != 0
}

func testTMode(m OpCode) bool {
	return luaP_opmodes[m]&(1<<7) != 0
}

var luaP_opnames = [NUM_OPCODES]string{
	"MOVE",
	"LOADK",
	"LOADBOOL",
	"LOADNIL",
	"GETUPVAL",
	"GETGLOBAL",
	"GETTABLE",
	"SETGLOBAL",
	"SETUPVAL",
	"SETTABLE",
	"NEWTABLE",
	"SELF",
	"ADD",
	"SUB",
	"MUL",
	"DIV",
	"MOD",
	"POW",
	"UNM",
	"NOT",
	"LEN",
	"CONCAT",
	"JMP",
	"EQ",
	"LT",
	"LE",
	"TEST",
	"TESTSET",
	"CALL",
	"TAILCALL",
	"RETURN",
	"FORLOOP",
	"FORPREP",
	"TFORLOOP",
	"SETLIST",
	"CLOSE",
	"CLOSURE",
	"VARARG",
}

var luaP_opmodes = [NUM_OPCODES]lu_byte{
	/*        T    A     B       C    mode     opcode             */
	opmode(0, 1, OpArgR, OpArgN, iABC),  /* OP_MOVE          */
	opmode(0, 1, OpArgK, OpArgN, iABx),  /* OP_LOADK         */
	opmode(0, 1, OpArgU, OpArgU, iABC),  /* OP_LOADBOOL      */
	opmode(0, 1, OpArgR, OpArgN, iABC),  /* OP_LOADNIL       */
	opmode(0, 1, OpArgU, OpArgN, iABC),  /* OP_GETUPVAL      */
	opmode(0, 1, OpArgK, OpArgN, iABx),  /* OP_GETGLOBAL     */
	opmode(0, 1, OpArgR, OpArgK, iABC),  /* OP_GETTABLE      */
	opmode(0, 0, OpArgK, OpArgN, iABx),  /* OP_SETGLOBAL     */
	opmode(0, 0, OpArgU, OpArgN, iABC),  /* OP_SETUPVAL      */
	opmode(0, 0, OpArgK, OpArgK, iABC),  /* OP_SETTABLE      */
	opmode(0, 1, OpArgU, OpArgU, iABC),  /* OP_NEWTABLE      */
	opmode(0, 1, OpArgR, OpArgK, iABC),  /* OP_SELF          */
	opmode(0, 1, OpArgK, OpArgK, iABC),  /* OP_ADD           */
	opmode(0, 1, OpArgK, OpArgK, iABC),  /* OP_SUB           */
	opmode(0, 1, OpArgK, OpArgK, iABC),  /* OP_MUL           */
	opmode(0, 1, OpArgK, OpArgK, iABC),  /* OP_DIV           */
	opmode(0, 1, OpArgK, OpArgK, iABC),  /* OP_MOD           */
	opmode(0, 1, OpArgK, OpArgK, iABC),  /* OP_POW           */
	opmode(0, 1, OpArgR, OpArgN, iABC),  /* OP_UNM           */
	opmode(0, 1, OpArgR, OpArgN, iABC),  /* OP_NOT           */
	opmode(0, 1, OpArgR, OpArgN, iABC),  /* OP_LEN           */
	opmode(0, 1, OpArgR, OpArgR, iABC),  /* OP_CONCAT        */
	opmode(0, 0, OpArgR, OpArgN, iAsBx), /* OP_JMP           */
	opmode(1, 0, OpArgK, OpArgK, iABC),  /* OP_EQ            */
	opmode(1, 0, OpArgK, OpArgK, iABC),  /* OP_LT            */
	opmode(1, 0, OpArgK, OpArgK, iABC),  /* OP_LE            */
	opmode(1, 1, OpArgR, OpArgU, iABC),  /* OP_TEST          */
	opmode(1, 1, OpArgR, OpArgU, iABC),  /* OP_TESTSET       */
	opmode(0, 1, OpArgU, OpArgU, iABC),  /* OP_CALL          */
	opmode(0, 1, OpArgU, OpArgU, iABC),  /* OP_TAILCALL      */
	opmode(0, 0, OpArgU, OpArgN, iABC),  /* OP_RETURN        */
	opmode(0, 1, OpArgR, OpArgN, iAsBx), /* OP_FORLOOP       */
	opmode(0, 1, OpArgR, OpArgN, iAsBx), /* OP_FORPREP       */
	opmode(1, 0, OpArgN, OpArgU, iABC),  /* OP_TFORLOOP      */
	opmode(0, 0, OpArgU, OpArgU, iABC),  /* OP_SETLIST       */
	opmode(0, 0, OpArgN, OpArgN, iABC),  /* OP_CLOSE         */
	opmode(0, 1, OpArgU, OpArgN, iABx),  /* OP_CLOSURE       */
	opmode(0, 1, OpArgU, OpArgN, iABC),  /* OP_VARARG        */
}

func opmode(t lu_byte, a, b, c lu_byte, m OpMode) lu_byte {
	return t<<7 | a<<6 | b<<4 | c<<2 | lu_byte(m)
}

const LFIELDS_PER_FLUSH = 50 /* number of list items to accumulate before a SETLIST instruction */
