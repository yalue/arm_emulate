package arm_emulate

import (
	"strings"
	"testing"
)

func TestUndefinedInstruction(t *testing.T) {
	raw := uint32(0xf7ffffff)
	// Undefined instructions should return an error, but still return a basic
	// instruction of some sort.
	n, e := ParseInstruction(raw)
	if e == nil {
		t.Fail()
	}
	if n.Raw() != raw {
		t.Fail()
	}
}

func TestSoftwareInterruptParse(t *testing.T) {
	raw := uint32(0x1f000000)
	n, e := ParseInstruction(raw)
	if e != nil {
		t.Fail()
	}
	if n.Raw() != raw {
		t.Fail()
	}
	if n.Condition().Condition() != 1 {
		t.Fail()
	}
	if !strings.Contains(n.String(), "swine") {
		t.Fail()
	}
}

func TestDPRInstruction(t *testing.T) {
	raw := uint32(0)
	n, e := ParseInstruction(raw)
	if e != nil {
		t.Fail()
	}
	if n.Raw() != raw {
		t.Fail()
	}
	if n.Condition().String() != "eq" {
		t.Fail()
	}
	if !strings.HasPrefix(n.String(), "and") {
		t.Fail()
	}
	if !strings.Contains(n.String(), "r0") {
		t.Fail()
	}
	raw = 0xe3530000
	n, e = ParseInstruction(raw)
	if e != nil {
		t.Fail()
	}
	if n.Condition().Condition() != 14 {
		t.Fail()
	}
	if !strings.HasPrefix(n.String(), "cmp") {
		t.Fail()
	}
	if !strings.Contains(n.String(), "r3") {
		t.Fail()
	}
	if !strings.Contains(n.String(), "0") {
		t.Fail()
	}
	raw = 0xe1a00200
	n, e = ParseInstruction(raw)
	if e != nil {
		t.Fail()
	}
	if !strings.HasPrefix(n.String(), "mov") {
		t.Fail()
	}
	if !strings.Contains(n.String(), "lsl") {
		t.Fail()
	}
	if !strings.Contains(n.String(), "4") {
		t.Fail()
	}
}

func TestBlockDataTransferInstruction(t *testing.T) {
	raw := uint32(0xe8bd8008)
	n, e := ParseInstruction(raw)
	if e != nil {
		t.Fail()
	}
	if !strings.HasPrefix(n.String(), "ldmfd") {
		t.Fail()
	}
	if !strings.Contains(n.String(), "pc") {
		t.Fail()
	}
	if !strings.Contains(n.String(), "sp!") {
		t.Fail()
	}
	if !strings.Contains(n.String(), "r3") {
		t.Fail()
	}
}
