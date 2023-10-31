package golua

import (
	"fmt"
	"testing"
)

func TestMASK1(t *testing.T) {

	s := fmt.Sprintf("%b", MASK1(10, 5))
	v := "111111111100000"
	if s != v {
		t.Errorf("want %s got %s", v, s)
	}
}

func TestMASK0(t *testing.T) {
	s := fmt.Sprintf("%b", MASK0(10, 5))
	v := "11111111111111111000000000011111"
	if s != v {
		t.Errorf("want %s got %s", v, s)
	}
}
