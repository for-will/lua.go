package golua

import (
	"testing"
)

func Test_l_strcmp(t *testing.T) {
	// t.Log(bytes.Compare([]byte("223421"), []byte("22346")))/* -1 */
}

func TestDoJump(t *testing.T) {

	var codes = []Instruction{
		10, 20, 11, 22, 33, 44, 55, 66, 987,
	}

	pc := &codes[0]

	var DoJump = func(n int) {
		pc = pc.Ptr(n)
	}

	for _, v := range codes {
		if v != *pc {
			t.Errorf("want %v got %v", v, *pc)
		}
		DoJump(1)
	}
}
