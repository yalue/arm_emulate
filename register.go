package arm_emulate

import (
	"fmt"
)

type ARMRegister uint8

func (r ARMRegister) String() string {
	if r < 13 {
		return fmt.Sprintf("r%d", r)
	}
	registerStrings := [...]string{"sp", "lr", "pc"}
	return registerStrings[r-13]
}
