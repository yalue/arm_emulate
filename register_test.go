package arm_emulate

import (
	"testing"
)

func TestNewARMRegister(t *testing.T) {
	r := ARMRegister(0)
	if r.String() != "r0" {
		t.Fail()
	}
	r = ARMRegister(13)
	if r.String() != "sp" {
		t.Fail()
	}
}
