package arm_emulate

import (
	"testing"
)

func TestNewARMDataProcessingOpcode(t *testing.T) {
	d := ARMDataProcessingOpcode(0)
	if d.String() != "and" {
		t.Fail()
	}
	d = ARMDataProcessingOpcode(9)
	if d.String() != "teq" {
		t.Fail()
	}
}
