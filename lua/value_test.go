package golua

import (
	"testing"
)

func TestStkId_ADD(t *testing.T) {
	l := make([]TValue, 10)

	top := &l[9]
	top.SetNumber(123.456)
	t.Log(top.Ptr(10).Ptr(-10).NumberValue())
}
