package golua

import "unsafe"

// Instruction 对应C类型`Instruction`
// type for virtual-machine instructions
// must be an unsigned with (at least) 4 bytes (see details in lopcodes.h)
type Instruction int32

func (i *Instruction) Ptr(n int) *Instruction {
	if n >= 0 {
		p := uintptr(unsafe.Pointer(i)) + uintptr(n)*unsafe.Sizeof(Instruction(0))
		return (*Instruction)(unsafe.Pointer(p))
	} else {
		p := uintptr(unsafe.Pointer(i)) - uintptr(-n)*unsafe.Sizeof(Instruction(0))
		return (*Instruction)(unsafe.Pointer(p))
	}
}
