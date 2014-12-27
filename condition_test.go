package arm_emulate

import (
	"testing"
)

func TestNewARMCondition(t *testing.T) {
	c := ARMCondition(0)
	if c.String() != "eq" {
		t.Fail()
	}
	c = ARMCondition(10)
	if c.String() != "ge" {
		t.Fail()
	}
}
