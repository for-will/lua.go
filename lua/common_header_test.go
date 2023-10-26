package golua

import (
	"testing"
)

func TestCommonHeader_ToClosure(t *testing.T) {
	lc := &CClosure{}
	var obj GCObject
	obj = lc
	t.Logf("%#v", obj.ToClosure().C())
}
