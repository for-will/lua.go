package golua

import (
	"math"
	"reflect"
	"testing"
	"unsafe"
)

func TestTable_HashNum(t1 *testing.T) {
	// t1.Log(NumInts)
	// var a [2]uint32
	// n := math.MaxInt64
	// b := []byte(n)
	// binary.LittleEndian.PutUint32()
	// n := math.MaxFloat64
	// n = 0
	// var tmp = make([]byte, 10)
	// nrb := binary.PutUvarint(tmp, math.Float64bits(-1))
	// t1.Log(tmp, nrb)
	// t1.Log(unsafe.Sizeof(n))
	// t1.Log(unsafe.Sizeof(uint64(10)))

	a := uint32(math.Float64bits(1))
	b := uint32(math.Float64bits(1) >> 32)
	t1.Logf("%#x -> H:%#x L:%#x", math.Float64bits(1), b, a)
	t1.Log(unsafe.Sizeof(uint(100)))
}

func TestAddrOfAny(t *testing.T) {
	s := TString{}
	var a interface{}
	a = &s
	t.Logf("%#X", reflect.ValueOf(a).Pointer())
	t.Logf("%p", a)

	p1 := &TString{}
	p2 := &TString{}
	var S TString
	p1 = &S
	t.Log(p1 == p2)
	var key = 0
	t.Log(uint(key - 1))
}

func TestTable_findIndex(t1 *testing.T) {
	l := make([]Node, 30)

	const sizeOfNode = unsafe.Sizeof(Node{})
	for i := 0; i < len(l); i++ {
		offset := uintptr(unsafe.Pointer(&l[i])) - uintptr(unsafe.Pointer(&l[0]))
		t1.Log(offset / sizeOfNode)
	}
}
