package golua

// Proto Function Prototypes
// 对应C结构体：`struct Proto`
type Proto struct {
	CommonHeader
	k               []TValue      /* constants used by the function */
	code            []Instruction //
	p               []*Proto      /* functions defined inside the function */
	lineInfo        []int         /* map from opcodes to source lines */
	locVars         []LocVar      //
	upValues        []*TString    /* upvalue names*/
	source          *TString
	sizeUpValues    int
	sizeK           int /* size of `k` */
	sizeCode        int
	sizeLineInfo    int
	sizeP           int /* size of `p` */
	sizeLocVars     int
	lineDefined     int
	lastLineDefined int
	gcList          *GCObject
	nups            int /* number of upvalues */
	numParams       int
	isVarArg        lu_byte
	maxStackSize    lu_byte
}
