package arm_emulate

import (
	"testing"
)

func TestNewARMDataProcessingOpcode(t *testing.T) {
	d := NewARMDataProcessingOpcode(0)
	if d.Value() != 0 {
		t.Fail()
	}
	if d.String() != "and" {
		t.Fail()
	}
	d = NewARMDataProcessingOpcode(9)
	if d.Value() != 9 {
		t.Fail()
	}
	if d.String() != "teq" {
		t.Fail()
	}
}
