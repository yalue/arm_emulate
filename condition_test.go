package arm_emulate

import (
	"testing"
)

func TestNewARMCondition(t *testing.T) {
	c := NewARMCondition(0)
	if c.String() != "eq" {
		t.Fail()
	}
	if c.Condition() != 0 {
		t.Fail()
	}
	c = NewARMCondition(10)
	if c.String() != "ge" {
		t.Fail()
	}
	if c.Condition() != 10 {
		t.Fail()
	}
}
