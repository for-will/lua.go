package golua

import (
	"bytes"
	"testing"
)

func Test_uHeader(t *testing.T) {

	var header = make([]byte, LUAC_HEADERSIZE)
	uHeader(header)
	if 0 != bytes.Compare(header[:LUAC_HEADERSIZE], []byte{27, 76, 117, 97, 81, 0, 1, 8, 8, 4, 8, 0}) {
		t.Error("invalid uHeader")
	}
}

func Test_loadState_LoadVar(t *testing.T) {
	s := &loadState{}
	var x float64 = 1.1
	s.LoadVar(&x)
	t.Log(x)
}

func Test_loadState_LoadVector(t *testing.T) {
	s := &loadState{}
	var x = make([]Instruction, 10)
	s.LoadVector(x, 10, 4)
	t.Log(x)
}
