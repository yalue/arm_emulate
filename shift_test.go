package arm_emulate

import (
	"testing"
)

func TestNewARMShift(t *testing.T) {
	s := NewARMShift(0x08)
	if s.Amount() != 1 {
		t.Fail()
	}
	if s.UseRegister() {
		t.Fail()
	}
	if s.ShiftType() != 0 {
		t.Fail()
	}
	if s.ShiftString() != "lsl" {
		t.Fail()
	}
	s = NewARMShift(0x47)
	if !s.UseRegister() {
		t.Fail()
	}
	if s.ShiftType() != 3 {
		t.Fail()
	}
	if s.ShiftString() != "ror" {
		t.Fail()
	}
	if s.Register().Register() != 4 {
		t.Fail()
	}
}
