package arm_emulate

import (
	"testing"
)

func TestNewARMRegister(t *testing.T) {
	r := NewARMRegister(0)
	if r.Register() != 0 {
		t.Fail()
	}
	if r.String() != "r0" {
		t.Fail()
	}
	r = NewARMRegister(13)
	if r.Register() != 13 {
		t.Fail()
	}
	if r.String() != "sp" {
		t.Fail()
	}
}
