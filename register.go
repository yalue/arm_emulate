package arm_emulate

import (
	"fmt"
)

// This interface is used for registers contained in instruction types. While
// raw numbers could also have been used, this interface is useful for
// sanity-checking that register numbers are valid and for producing the string
// mnemonic for the register during disassembly.
type ARMRegister interface {
	fmt.Stringer
	Register() uint8
}

type basicARMRegister struct {
	number uint8
}

func (r *basicARMRegister) String() string {
	if r.number < 13 {
		return fmt.Sprintf("r%d", r.number)
	}
	registerStrings := [...]string{"sp", "lr", "pc"}
	return registerStrings[r.number-13]
}

func (r *basicARMRegister) Register() uint8 {
	return r.number
}

func NewARMRegister(number uint8) ARMRegister {
	return &basicARMRegister{number & 0xf}
}
