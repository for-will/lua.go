package golua

import (
	"os"
	"testing"
)

func TestFILE_ungetc(t *testing.T) {
	f, _ := fopen("file_test.go", os.O_RDONLY)
	var data []byte
	for !f.EOF() {
		b, ok := f.getc()
		if !ok {
			t.Error("getc failed")
		}
		data = append(data, b)
	}

	for i := len(data) - 1; i >= 0; i-- {
		f.ungetc(data[i])
	}

	for i := 0; !f.EOF(); i++ {
		b, ok := f.getc()
		if !ok {
			t.Error("getc failed")
		}
		if b != data[i] {
			t.Errorf("want %v got %v", data[i], b)
		}
	}

	for i := len(data) - 1; i >= 0; i-- {
		f.ungetc(data[i])
	}
}

func TestFILE_fread(t *testing.T) {
	f, _ := fopen("file_test.go", os.O_RDONLY)
	var data []byte
	for !f.EOF() {
		b, ok := f.getc()
		if !ok {
			t.Error("getc failed")
		}
		data = append(data, b)
	}

	f2, _ := fopen("file_test.go", os.O_RDONLY)
	var buf [31]byte
	j := 0
	for !f2.EOF() {
		n := f2.fread(buf[:])
		for i := 0; i < n; i++ {
			if buf[i] != data[j] {
				t.Errorf("want %v got %v", data[j], data[i])
			}
			j++
		}
	}
	if j != len(data) {
		t.Errorf("want %v got %v", len(data), j)
	}
}
