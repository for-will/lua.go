package golua

import (
	"bytes"
	"unsafe"
)

// TString 字符串类型
// Bytes - 我们直接分配内存，使用go的垃圾回收机制管理内存，而不用手动管理内存
type TString struct {
	CommonHeader
	Reserved lu_byte // 为1表示是Lua中的关键字
	Hash     uint64
	Len      uint64
	Bytes    []byte
}

func (s *TString) ToString() *TString {
	return s
}

func (s *TString) GetBytes() []byte {
	return s.Bytes
}

func (s *TString) GetStr() []byte {
	return s.Bytes
}

func (s *TString) Fix() {
	s.marked |= 1 << FIXEDBIT
}

type StringTable struct {
	Hash []GCObject
	NUse uint64
	Size uint64
}

func (L *LuaState) LuaSResize(newSize uint64) {

	// todo:
	// if (G(L)->gcstate == GCSsweepstring)
	// return;  /* cannot resize during GC traverse */

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
			next := p.Next() // save next
			h := p.ToString().Hash
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

// newlstr 创建新的字符串
// str 字符切片
// l 长度
// h 字符hash值
func (L *LuaState) newlstr(str []byte, l uint64, h uint64) *TString {
	if l+1 > uint64(MAX_SIZET)-uint64(unsafe.Sizeof(TString{})) {
		L.LuaMTooBig()
	}
	ts := &TString{
		CommonHeader: CommonHeader{
			next:   nil,
			tt:     LUA_TSTRING,
			marked: L.G().LuaCWhite(),
		},
		Reserved: 0,
		Hash:     h,
		Len:      l,
		Bytes:    make([]byte, l+1),
	}
	copy(ts.Bytes, str[:l])
	ts.Bytes[l] = 0 // ending 0
	tb := L.G().StrT
	h = LMod(h, tb.Size)
	ts.next = tb.Hash[h] // chain new entry
	tb.Hash[h] = ts
	tb.NUse++
	if tb.NUse > tb.Size && tb.Size <= MAX_INT/2 {
		L.LuaSResize(tb.Size * 2)
	}
	return ts
}

// SNewLStr ->
// TString *luaS_newlstr (lua_State *L, const char *str, size_t l)
func (L *LuaState) SNewLStr(str []byte, l uint64) *TString {
	h := l
	step := (l >> 5) + 1
	for l1 := l; l1 >= step; l1 -= step {
		// compute hash
		h = h ^ ((h << 5) + (h >> 2)) + uint64(str[l1-1])
	}
	o := L.G().StrT.Hash[LMod(h, L.G().StrT.Size)]
	for ; o != nil; o = o.Next() {
		ts := o.ToString()
		if ts.Len == l && bytes.Compare(str[:l], ts.GetStr()) == 0 {
			// todo:
			// /* string may be dead */
			// if (isdead(G(L), o)) changewhite(o);
			return ts
		}
	}
	return L.newlstr(str, l, h)
}

func (L *LuaState) SNew(b []byte) *TString {
	return L.SNewLStr(b, uint64(len(b)))
}

func LMod(s, size uint64) uint64 {
	CheckExp(size&(size-1) == 0)
	return s & (size - 1)
}
