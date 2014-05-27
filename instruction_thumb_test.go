package arm_emulate

import (
	"strings"
	"testing"
)

func TestPushInstruction(t *testing.T) {
	raw := uint16(0xb580)
	n, e := ParseTHUMBInstruction(raw)
	if e != nil {
		t.Fail()
	}
	if n.Raw() != raw {
		t.Fail()
	}
	if !strings.HasPrefix(n.String(), "push") {
		t.Fail()
	}
	if !strings.Contains(n.String(), "r7") {
		t.Fail()
	}
	if !strings.Contains(n.String(), "lr") {
		t.Fail()
	}
	raw = uint16(0xbd80)
	n, e = ParseTHUMBInstruction(raw)
	if e != nil {
		t.Fail()
	}
	if n.Raw() != raw {
		t.Fail()
	}
	if !strings.HasPrefix(n.String(), "pop") {
		t.Fail()
	}
	if !strings.Contains(n.String(), "pc") {
		t.Fail()
	}
	if !strings.Contains(n.String(), "r7") {
		t.Fail()
	}
}

func TestAddToStackPointerInstruction(t *testing.T) {
	raw := uint16(0xb082)
	n, e := ParseTHUMBInstruction(raw)
	if e != nil {
		t.Fail()
	}
	if n.Raw() != raw {
		t.Fail()
	}
	if !strings.Contains(n.String(), "sp") {
		t.Fail()
	}
	if strings.HasPrefix(n.String(), "add") {
		if !strings.Contains(n.String(), "-8") {
			t.Fail()
		}
	} else if !strings.Contains(n.String(), "sub") {
		t.Fail()
	}
}
