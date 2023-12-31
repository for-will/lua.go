package golua

import (
	"bytes"
	"luar/lua/mem"
	"unsafe"
)

// TString 字符串类型
// Bytes - 我们直接分配内存，使用go的垃圾回收机制管理内存，而不用手动管理内存
type TString struct {
	CommonHeader
	Reserved lu_byte // 为1表示是Lua中的关键字
	Hash     uint64
	Len      int
	Bytes    []byte
}

// func (s *TString) ToString() *TString {
// 	return s
// }

func (s *TString) GetBytes() []byte {
	return s.Bytes
}

func (s *TString) GetStr() []byte {
	return s.Bytes
}

// Fix
// 对应C函数：`luaS_fix(s)'
func (s *TString) Fix() {
	s.marked |= 1 << FIXEDBIT
}

type StringTable struct {
	Hash  []GCObject
	NrUse uint64
	Size  uint64
}

// sResize
// 对应C函数：`void luaS_resize (lua_State *L, int newsize)'
func (L *LuaState) sResize(newSize uint64) {

	// todo: if (G(L)->gcstate == GCSsweepstring)
	// todo: return;  /* cannot resize during GC traverse */

	// newhash = luaM_newvector(L, newsize, GCObject *);
	newHash := make([]GCObject, newSize)
	tb := L.G().StrT // tb = &G(L)->strt;
	for i := uint64(0); i < newSize; i++ {
		newHash[i] = nil
	}
	/* rehash */
	for i := 0; i < int(tb.Size); i++ {
		p := tb.Hash[i]
		for p != nil {
			next := p.GetNext() // save next
			h := p.ToTString().Hash
			h1 := LMod(h, newSize) // new position
			LuaAssert((h % newSize) == LMod(h, newSize))
			p.SetNext(newHash[h1]) // chain it
			newHash[h1] = p
			p = next
		}
	}
	// luaM_freearray(L, tb->hash, tb->size, TString *);
	tb.Size = newSize
	tb.Hash = newHash
}

// newStr 创建新的字符串
// str 字符切片；
// h 字符hash值；
// NOTE: 这里保存的字符串没有'\0'结尾。
// 对应C函数：`static TString *newlstr (lua_State *L, const char *str, size_t l, unsigned int h)'
func (L *LuaState) newStr(str []byte, h uint64) *TString {
	var l = len(str)
	if l > int(MAX_SIZET)-int(unsafe.Sizeof(TString{})) {
		// L.MemTooBig()
		mem.ErrTooBig(L)
	}
	ts := &TString{
		CommonHeader: CommonHeader{
			next:   nil,
			tt:     LUA_TSTRING,
			marked: L.G().cWhite(),
		},
		Reserved: 0,
		Hash:     h,
		Len:      l,
		Bytes:    make([]byte, l),
	}
	copy(ts.Bytes, str)
	// ts.Bytes[l] = 0 /* ending 0 */
	var tb = L.G().StrT /* global string table */
	h = LMod(h, tb.Size)
	ts.next = tb.Hash[h] /* chain new entry */
	tb.Hash[h] = ts
	tb.NrUse++
	if tb.NrUse > tb.Size && tb.Size <= MAX_INT/2 {
		L.sResize(tb.Size * 2)
	}
	return ts
}

// sNewStr
// 对应C函数：`TString *luaS_newlstr (lua_State *L, const char *str, size_t l)'
func (L *LuaState) sNewStr(str []byte) *TString {
	var (
		l    = len(str)
		h    = uint64(l)    /* seed */
		step = (l >> 5) + 1 /* if string is too long, don't hash all its chars */
	)
	for l1 := l; l1 >= step; l1 -= step { /* compute hash */
		h = h ^ ((h << 5) + (h >> 2)) + uint64(str[l1-1])
	}
	var o = L.G().StrT.Hash[LMod(h, L.G().StrT.Size)]
	for ; o != nil; o = o.GetNext() {
		ts := o.ToTString()
		if ts.Len == l && bytes.Compare(str, ts.GetStr()) == 0 {
			// todo: if (isdead(G(L), o)) changewhite(o);
			return ts
		}
	}
	return L.newStr(str, h)
}

func (L *LuaState) sNew(b []byte) *TString {
	return L.sNewStr(b)
}

func LMod(s, size uint64) uint64 {
	CheckExp(size&(size-1) == 0)
	return s & (size - 1)
}
