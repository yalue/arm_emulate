package arm_emulate

import (
	"fmt"
)

var registerStrings = [...]string{"r0", "r1", "r2", "r3", "r4", "r5", "r6",
	"r7", "r8", "r9", "r10", "r11", "r12", "sp", "lr", "pc"}

type ARMRegister interface {
	fmt.Stringer
	Register() uint8
}

type basicARMRegister struct {
	number uint8
}

func (r *basicARMRegister) String() string {
	return registerStrings[r.number&0xf]
}

func (r *basicARMRegister) Register() uint8 {
	return r.number
}

func NewARMRegister(number uint8) ARMRegister {
	return &basicARMRegister{number & 0xf}
}
