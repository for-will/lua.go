package golua

import (
	"bytes"
	"reflect"
	"unsafe"
)

const (
	LUAC_VERSION    = 0x51 /* for header of binary files -- this is Lua 5.1 */
	LUAC_FORMAT     = 0    /* for header of binary files -- this is official format */
	LUAC_HEADERSIZE = 12   /* size of header of binary files */
)

// 对应C结构体：`struct LoadState'
type loadState struct {
	L    *LuaState
	Z    *ZIO
	b    *MBuffer
	name []byte
}

// IF 对应C函数：`IF(c,s)'
func (S *loadState) IF(c bool, s string) {
	if c {
		S.error(s)
	}
}

// 对应C函数：`static void error(LoadState* S, const char* why)'
func (S *loadState) error(why string) {
	S.L.oPushFString("%s: %s in precompiled chunk", S.name, why)
	S.L.dThrow(LUA_ERRSYNTAX)
}

// LoadMem
// 对应C函数：`LoadMem(S,b,n,size)'
func (S *loadState) LoadMem(b []byte, n int, size int) {
	S.LoadBlock(b[:n*size])
}

// LoadByte
// 对应C函数：`LoadByte(S)'
func (S *loadState) LoadByte() lu_byte {
	return lu_byte(S.LoadChar())
}

// LoadVar
// 对应C函数：`LoadVar(S,x)'
func (S *loadState) LoadVar(x interface{}) {
	// 应该考虑使用泛型？？
	size := int(reflect.TypeOf(x).Elem().Size())
	var buf = make([]byte, size)
	S.LoadMem(buf, 1, size)
	var p = reflect.ValueOf(x).Pointer()
	for i := 0; i < size; i++ {
		*(*byte)(unsafe.Pointer(p + uintptr(i))) = buf[i]
	}
}

// LoadVector
// 对应C函数：`LoadVector(S,b,n,size)'
func (S *loadState) LoadVector(b interface{}, n int, size int) {
	totalBytes := n * size
	buf := make([]byte, totalBytes)
	S.LoadMem(buf, n, size)
	var p = reflect.ValueOf(b).Index(0).Addr().Pointer()
	for i := 0; i < totalBytes; i++ {
		*(*byte)(unsafe.Pointer(p + uintptr(i))) = buf[i]
	}
}

// LoadBlock
// 对应C函数：`static void LoadBlock(LoadState* S, void* b, size_t size)'
func (S *loadState) LoadBlock(b []byte) {
	r := S.Z.Read(b, len(b))
	S.IF(r != 0, "unexpected end")
}

// LoadChar
// 对应C函数：`static int LoadChar(LoadState* S)'
func (S *loadState) LoadChar() int {
	var x byte
	S.LoadVar(&x)
	return int(x)
}

// LoadInt
// 对应C函数：`static int LoadInt(LoadState* S)'
func (S *loadState) LoadInt() int {
	var x int
	S.LoadVar(&x)
	S.IF(x < 0, "bad integer")
	return x
}

// LoadNumber
// 对应C函数：`static lua_Number LoadNumber(LoadState* S)'
func (S *loadState) LoadNumber() LuaNumber {
	var x LuaNumber
	S.LoadVar(&x)
	return x
}

// LoadString
// 对应C函数：`static TString* LoadString(LoadState* S)'
func (S *loadState) LoadString() *TString {
	var size int
	S.LoadVar(&size)
	if size == 0 {
		return nil
	} else {
		s := S.b.OpenSpace(size)
		S.LoadBlock(s[:size])
		return S.L.sNewLStr(s[:size-1]) /* remove trailing '\0' */
	}
}

// LoadCode
// 对应C函数：`static void LoadCode(LoadState* S, Proto* f)'
func (S *loadState) LoadCode(f *Proto) {
	var n = S.LoadInt()
	f.code = make([]Instruction, n)
	f.sizeCode = n
	S.LoadVector(f.code, n, int(unsafe.Sizeof(Instruction(0))))
}

// LoadConstants
// 对应C函数：`static void LoadConstants(LoadState* S, Proto* f)'
func (S *loadState) LoadConstants(f *Proto) {
	n := S.LoadInt()
	f.k = make([]TValue, n)
	f.sizeK = n
	for i := 0; i < n; i++ {
		f.k[i].SetNil()
	}
	for i := 0; i < n; i++ {
		o := &f.k[i]
		t := S.LoadChar()
		switch ttype(t) {
		case LUA_TNIL:
			o.SetNil()
		case LUA_TBOOLEAN:
			o.SetBoolean(S.LoadChar() != 0)
		case LUA_TNUMBER:
			o.SetNumber(S.LoadNumber())
		case LUA_TSTRING:
			o.SetString(S.L, S.LoadString())
		default:
			S.error("bad constant")
		}
	}
	n = S.LoadInt()
	f.p = make([]*Proto, n)
	f.sizeP = n
	for i := 0; i < n; i++ {
		f.p[i] = S.LoadFunction(f.source)
	}
}

// LoadDebug
// 对应C函数：`static void LoadDebug(LoadState* S, Proto* f)'
func (S *loadState) LoadDebug(f *Proto) {
	n := S.LoadInt()
	f.lineInfo = make([]int, n)
	f.sizeLineInfo = n
	S.LoadVector(f.lineInfo, n, int(unsafe.Sizeof(int(0))))
	n = S.LoadInt()
	f.locVars = make([]LocVar, n)
	f.sizeLocVars = n
	for i := 0; i < n; i++ {
		f.locVars[i].varName = S.LoadString()
		f.locVars[i].startPc = S.LoadInt()
		f.locVars[i].endPc = S.LoadInt()
	}
	n = S.LoadInt()
	f.upValues = make([]*TString, n)
	f.sizeUpValues = n
	for i := 0; i < n; i++ {
		f.upValues[i] = S.LoadString()
	}
}

// LoadFunction
// 对应C函数：`static Proto* LoadFunction(LoadState* S, TString* p)'
func (S *loadState) LoadFunction(p *TString) *Proto {
	S.L.nCCalls++
	if S.L.nCCalls >= LUAI_MAXCCALLS {
		S.error("code too deep")
	}
	f := S.L.fNewProto()
	S.L.Top().SetProto(S.L, f)
	S.L.IncTop()
	f.source = S.LoadString()
	if f.source == nil {
		f.source = p
	}
	f.lineDefined = S.LoadInt()
	f.lastLineDefined = S.LoadInt()
	f.nUps = S.LoadChar()
	f.numParams = S.LoadChar()
	f.isVarArg = S.LoadByte()
	f.maxStackSize = S.LoadByte()
	S.LoadCode(f)
	S.LoadConstants(f)
	S.LoadDebug(f)
	S.IF(gCheckCode(f) != 0, "bad code")
	S.L.top++
	S.L.nCCalls--
	return f
}

// LoadHeader
// 对应C函数：`static void LoadHeader(LoadState* S)'
func (S *loadState) LoadHeader() {
	var h, s [LUAC_HEADERSIZE]byte
	uHeader(h[:])
	S.LoadBlock(s[:])
	S.IF(bytes.Compare(h[:], s[:]) != 0, "bad header")
}

// uUndump load precompiled chuck
// 对应C函数：`Proto* luaU_undump (lua_State* L, ZIO* Z, Mbuffer* buff, const char* name)'
func (L *LuaState) uUndump(Z *ZIO, buff *MBuffer, name []byte) *Proto {
	var S = loadState{
		L: L,
		Z: Z,
		b: buff,
	}
	if name[0] == '@' || name[0] == '=' {
		S.name = name[1:]
	} else if name[0] == LUA_SIGNATURE[0] {
		S.name = []byte("binary string")
	} else {
		S.name = name
	}
	S.LoadHeader()
	return S.LoadFunction(L.sNewLiteral("=?"))
}

// make header
// 对应C函数：`void luaU_header (char* h)'
func uHeader(h []byte) {
	var x = 1
	copy(h, LUA_SIGNATURE)
	h = h[len(LUA_SIGNATURE):len(LUA_SIGNATURE)]
	h = append(h,
		byte(LUAC_VERSION),
		byte(LUAC_FORMAT),
		*(*byte)(unsafe.Pointer(&x)), /* endianness */
		byte(unsafe.Sizeof(int(0))),
		byte(unsafe.Sizeof(uintptr(0))),
		byte(unsafe.Sizeof(Instruction(0))),
		byte(unsafe.Sizeof(LuaNumber(0))),
	)
	if LuaNumber(0.5) == 0 { /* is lua_Number integral*/
		_ = append(h, 1)
	} else {
		_ = append(h, 0)
	}
}
