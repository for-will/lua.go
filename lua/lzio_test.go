package golua

import "testing"

func TestMBuffer_Resize(t *testing.T) {
	buf := &MBuffer{}
	buf.OpenSpace(100)
	buf.Free()
	if buf.n != 0 || cap(buf.buffer) != 0 {
		t.Error("free MBuffer error")
	}
}
