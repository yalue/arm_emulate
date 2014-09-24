package arm_emulate

import (
	"fmt"
)

// This is a generic coprocessor interface. All coprocessors associated with an
// ARMProcessor that match the number in the opcode will have either their
// Operation, DataTransfer or RegisterTransfer methods called when the
// corresponding ARM instruction is emulated.
type ARMCoprocessor interface {
	Operation(p ARMProcessor, raw uint32) error
	DataTransfer(p ARMProcessor, raw, address uint32) error
	RegisterTransfer(p ARMProcessor, raw uint32, rd ARMRegister,
		load bool) error
	Number() uint8
}

// A simple coprocessor for testing. It's single register holds the last data
// transferred to it, and its single operation increments its register.
type simpleCounterCoprocessor struct {
	coprocNumber uint8
	register     uint32
}

func (c *simpleCounterCoprocessor) Number() uint8 {
	return c.coprocNumber
}

func (c *simpleCounterCoprocessor) Operation(p ARMProcessor, raw uint32) error {
	if uint8((raw>>8)&0xf) != c.coprocNumber {
		return nil
	}
	c.register++
	return nil
}

func (c *simpleCounterCoprocessor) DataTransfer(p ARMProcessor, raw,
	address uint32) error {
	if uint8((raw>>8)&0xf) != c.coprocNumber {
		return nil
	}
	load := (raw & 0x100000) != 0
	if load {
		value, e := p.GetMemoryInterface().ReadMemoryWord(address)
		if e != nil {
			return fmt.Errorf("Coprocessor error reading: %s", e)
		}
		c.register = value
	} else {
		e := p.GetMemoryInterface().WriteMemoryWord(address, c.register)
		if e != nil {
			return fmt.Errorf("Coprocessor error writing: %s", e)
		}
	}
	return nil
}

func (c *simpleCounterCoprocessor) RegisterTransfer(p ARMProcessor, raw uint32,
	rd ARMRegister, load bool) error {
	if uint8((raw>>8)&0xf) != c.coprocNumber {
		return nil
	}
	loadFromCoprocessor := (raw & 0x100000) != 0
	if loadFromCoprocessor {
		p.SetRegister(rd, c.register)
	} else {
		value, _ := p.GetRegister(rd)
		c.register = value
	}
	return nil
}

func NewTestStorageCoprocessor(number uint8) ARMCoprocessor {
	var c simpleCounterCoprocessor
	c.coprocNumber = number
	c.register = 0
	return &c
}
