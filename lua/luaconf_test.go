package golua

import (
	"strconv"
	"testing"
)

func TestNumberToStr(t *testing.T) {
	t.Log(NumberToStr(12311))
	t.Log(strconv.FormatFloat(-1.5, 'g', 14, 64))
}
