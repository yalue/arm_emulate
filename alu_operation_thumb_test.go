package arm_emulate

import (
	"testing"
)

func TestNewALUOpcodeTHUMB(t *testing.T) {
	o := NewALUOpcodeTHUMB(3)
	if o.Value() != 3 {
		t.Fail()
	}
	if o.String() != "lsr" {
		t.Fail()
	}
}
