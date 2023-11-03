package golua

import "testing"

func TestLexState_txtToken(t *testing.T) {
	var b = []byte("1234567890")

	b[5] = 0
	t.Log(string(b[:6]))
}
