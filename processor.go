package arm_emulate

import (
	"fmt"
)

const (
	userMode       uint8 = 0x10
	fiqMode        uint8 = 0x11
	irqMode        uint8 = 0x12
	supervisorMode uint8 = 0x13
	abortMode      uint8 = 0x17
	undefinedMode  uint8 = 0x1b
	systemMode     uint8 = 0x1f
)

func isValidMode(mode uint8) bool {
	return (mode == userMode) || (mode == systemMode) || (mode == fiqMode) ||
		(mode == irqMode) || (mode == supervisorMode) || (mode == abortMode) ||
		(mode == undefinedMode)
}

// The generic processor interface through which emulation functions should be
// implemented.
type ARMProcessor interface {
	// The functions for getting and setting registers take either a plain
	// register number (0-15) or an ARMRegister object. For now, the error
	// value can usually be ignored, but if this leads to problems, it should
	// be changed everywhere.
	GetRegister(register ARMRegister) (uint32, error)
	GetRegisterNumber(number uint8) (uint32, error)
	SetRegister(register ARMRegister, value uint32) error
	SetRegisterNumber(number uint8, value uint32) error
	// These operations always use the user-mode register bank.
	GetUserRegisterNumber(number uint8) (uint32, error)
	SetUserRegisterNumber(number uint8, value uint32) error
	// This returns the interface through which memory associated with this
	// processor may be modified.
	GetMemoryInterface() ARMMemory
	// The following functions may be used to set and access the processor's
	// state.
	GetMode() uint8
	SetMode(mode uint8) error
	Negative() bool
	Zero() bool
	Carry() bool
	Overflow() bool
	SetNegative(negative bool)
	SetZero(zero bool)
	SetCarry(carry bool)
	SetOverflow(overflow bool)
	FIQDisabled() bool
	IRQDisabled() bool
	THUMBMode() bool
	SetTHUMBMode(thumbMode bool) error
	// These functions access the raw status register value. It is preferable,
	// however, to use the preceding functions to modify specific bits.
	GetCPSR() (uint32, error)
	GetSPSR() (uint32, error)
	SetCPSR(value uint32) error
	SetSPSR(value uint32) error
	// AddCoprocessor may be used to associate an object implementing the
	// ARMCoprocessor interface with the processor. GetCoprocessors() is
	// primarily needed during emulation.
	AddCoprocessor(coprocessor ARMCoprocessor) error
	GetCoprocessors() []ARMCoprocessor
	// This prints the disassembly of instruction that will be executed on the
	// next call to RunNextInstruction()
	PendingInstructionString() string
	// This emulates a single instruction.
	RunNextInstruction() error
}

type basicARMProcessor struct {
	memory                        ARMMemory
	coprocessors                  []ARMCoprocessor
	currentRegisters              [16]uint32
	currentStatusRegister         uint32
	fiqRegisters                  [7]uint32
	supervisorRegisters           [2]uint32
	abortRegisters                [2]uint32
	irqRegisters                  [2]uint32
	undefinedRegisters            [2]uint32
	fiqSavedStatusRegister        uint32
	supervisorSavedStatusRegister uint32
	abortSavedStatusRegister      uint32
	irqSavedStatusRegister        uint32
	undefinedSavedStatusRegister  uint32
}

func (p *basicARMProcessor) GetMode() uint8 {
	return uint8(p.currentStatusRegister & 0x1f)
}

func (p *basicARMProcessor) SetMode(mode uint8) error {
	switchingToUser := false
	switch mode {
	case userMode:
		switchingToUser = true
		break
	case fiqMode, irqMode, supervisorMode, abortMode, undefinedMode,
		systemMode:
		break
	default:
		return fmt.Errorf("Invalid mode provided: 0x%02x.", mode)
	}
	oldStatus := p.currentStatusRegister
	p.currentStatusRegister = (oldStatus & 0xffffffe0) | uint32(mode)
	if !switchingToUser {
		e := p.SetSPSR(oldStatus)
		if e != nil {
			return fmt.Errorf("Failed writing SPSR in new mode: %s", e)
		}
	}
	return nil
}

func (p *basicARMProcessor) Negative() bool {
	return (p.currentStatusRegister & 0x80000000) != 0
}

func (p *basicARMProcessor) Zero() bool {
	return (p.currentStatusRegister & 0x40000000) != 0
}

func (p *basicARMProcessor) Carry() bool {
	return (p.currentStatusRegister & 0x20000000) != 0
}

func (p *basicARMProcessor) Overflow() bool {
	return (p.currentStatusRegister & 0x10000000) != 0
}

func (p *basicARMProcessor) SetNegative(negative bool) {
	if negative {
		p.currentStatusRegister |= 0x80000000
	} else {
		p.currentStatusRegister &= 0x7fffffff
	}
}

func (p *basicARMProcessor) SetZero(zero bool) {
	if zero {
		p.currentStatusRegister |= 0x40000000
	} else {
		p.currentStatusRegister &= 0xbfffffff
	}
}

func (p *basicARMProcessor) SetCarry(carry bool) {
	if carry {
		p.currentStatusRegister |= 0x20000000
	} else {
		p.currentStatusRegister &= 0xdfffffff
	}
}

func (p *basicARMProcessor) SetOverflow(overflow bool) {
	if overflow {
		p.currentStatusRegister |= 0x10000000
	} else {
		p.currentStatusRegister &= 0xefffffff
	}
}

func (p *basicARMProcessor) THUMBMode() bool {
	return (p.currentStatusRegister & 0x00000020) != 0
}

func (p *basicARMProcessor) SetTHUMBMode(thumbMode bool) error {
	if thumbMode {
		p.currentStatusRegister |= 0x00000020
	} else {
		p.currentStatusRegister &= 0xffffffdf
	}
	return nil
}

func (p *basicARMProcessor) FIQDisabled() bool {
	return (p.currentStatusRegister & 0x00000040) != 0
}

func (p *basicARMProcessor) IRQDisabled() bool {
	return (p.currentStatusRegister & 0x00000080) != 0
}

func (p *basicARMProcessor) GetRegister(register ARMRegister) (uint32, error) {
	return p.GetRegisterNumber(register.Register())
}

func (p *basicARMProcessor) GetRegisterNumber(number uint8) (uint32, error) {
	if number > 15 {
		return 0, fmt.Errorf("Error! Trying to read register %d?", number)
	}
	if number == 15 {
		return p.currentRegisters[number], nil
	}
	if number < 8 {
		return p.currentRegisters[number], nil
	}
	mode := p.GetMode()
	if (mode == userMode) || (mode == systemMode) {
		return p.currentRegisters[number], nil
	}
	if mode == fiqMode {
		return p.fiqRegisters[number-8], nil
	}
	if number < 13 {
		return p.currentRegisters[number], nil
	}
	number -= 13
	switch mode {
	case supervisorMode:
		return p.supervisorRegisters[number], nil
	case abortMode:
		return p.abortRegisters[number], nil
	case irqMode:
		return p.irqRegisters[number], nil
	case undefinedMode:
		return p.undefinedRegisters[number], nil
	}
	// Should be unreachable
	return 0, fmt.Errorf("Error getting register %d value", number)
}

func (p *basicARMProcessor) SetRegister(register ARMRegister,
	value uint32) error {
	return p.SetRegisterNumber(register.Register(), value)
}

func (p *basicARMProcessor) SetRegisterNumber(number uint8,
	value uint32) error {
	if number > 15 {
		return fmt.Errorf("Error! Trying to write register %d?", number)
	}
	if number == 15 {
		p.currentRegisters[number] = value
		return nil
	}
	if number < 8 {
		p.currentRegisters[number] = value
		return nil
	}
	mode := p.GetMode()
	if (mode == userMode) || (mode == systemMode) {
		p.currentRegisters[number] = value
		return nil
	}
	if mode == fiqMode {
		p.fiqRegisters[number-8] = value
		return nil
	}
	if number < 13 {
		p.currentRegisters[number] = value
		return nil
	}
	number -= 13
	switch mode {
	case supervisorMode:
		p.supervisorRegisters[number] = value
	case abortMode:
		p.abortRegisters[number] = value
	case irqMode:
		p.irqRegisters[number] = value
	case undefinedMode:
		p.undefinedRegisters[number] = value
	}
	return nil
}

func (p *basicARMProcessor) GetUserRegisterNumber(number uint8) (uint32,
	error) {
	if number > 15 {
		return 0, fmt.Errorf("Can't get user-bank r%d", number)
	}
	return p.currentRegisters[number], nil
}

func (p *basicARMProcessor) SetUserRegisterNumber(number uint8,
	value uint32) error {
	if number > 15 {
		return fmt.Errorf("Can't set user-bank r%d", number)
	}
	p.currentRegisters[number] = value
	return nil
}

func (p *basicARMProcessor) GetMemoryInterface() ARMMemory {
	return p.memory
}

func (p *basicARMProcessor) GetCPSR() (uint32, error) {
	return p.currentStatusRegister, nil
}

func (p *basicARMProcessor) GetSPSR() (uint32, error) {
	mode := p.GetMode()
	switch mode {
	case fiqMode:
		return p.fiqSavedStatusRegister, nil
	case supervisorMode:
		return p.supervisorSavedStatusRegister, nil
	case abortMode:
		return p.abortSavedStatusRegister, nil
	case irqMode:
		return p.irqSavedStatusRegister, nil
	case undefinedMode:
		return p.undefinedSavedStatusRegister, nil
	}
	return 0, fmt.Errorf("Mode 0x%02x doesn't have a SPSR", mode)
}

// TODO: Protection on mode changes! Don't allow switching modes from user mode
func (p *basicARMProcessor) SetCPSR(value uint32) error {
	current, e := p.GetCPSR()
	if e != nil {
		return e
	}
	oldMode := p.GetMode()
	newMode := uint8(value & 0x1f)
	if oldMode != newMode {
		e = p.SetMode(newMode)
		if e != nil {
			return e
		}
	}
	if oldMode == userMode {
		value = (value & 0xf0000000) | (current & 0x0fffffff)
	}
	p.currentStatusRegister = value
	return nil
}

func (p *basicARMProcessor) SetSPSR(value uint32) error {
	mode := p.GetMode()
	switch mode {
	case fiqMode:
		p.fiqSavedStatusRegister = value
		return nil
	case supervisorMode:
		p.supervisorSavedStatusRegister = value
		return nil
	case abortMode:
		p.abortSavedStatusRegister = value
		return nil
	case irqMode:
		p.irqSavedStatusRegister = value
		return nil
	case undefinedMode:
		p.undefinedSavedStatusRegister = value
		return nil
	}
	return fmt.Errorf("Mode 0x%02x doesn't have a SPSR", mode)
}

func (p *basicARMProcessor) AddCoprocessor(c ARMCoprocessor) error {
	p.coprocessors = append(p.coprocessors, c)
	return nil
}

func (p *basicARMProcessor) GetCoprocessors() []ARMCoprocessor {
	return p.coprocessors
}

func (p *basicARMProcessor) PendingInstructionString() string {
	pc, e := p.GetRegisterNumber(15)
	if e != nil {
		return fmt.Sprintf("Error fetching address: %s", e)
	}
	if p.THUMBMode() {
		raw, e := p.GetMemoryInterface().ReadMemoryHalfword(pc)
		if e != nil {
			return fmt.Sprintf("%08x: Error: %s", pc, e)
		}
		instruction, e := ParseTHUMBInstruction(raw)
		if e != nil {
			return fmt.Sprintf("%08x: %04x Error: %s", pc, raw, e)
		}
		return fmt.Sprintf("%08x: %04x %s", pc, raw, instruction)
	}
	raw, e := p.GetMemoryInterface().ReadMemoryWord(pc)
	if e != nil {
		return fmt.Sprintf("%08x: Error: %s", pc, e)
	}
	instruction, e := ParseInstruction(raw)
	if e != nil {
		return fmt.Sprintf("%08x: %08x Error: %s", pc, raw, e)
	}
	return fmt.Sprintf("%08x: %08x %s", pc, raw, instruction)
}

// This function will fetch an instruction, *increment pc*, then emulate the
// instruction. Therefore, pc will contain the address of the instruction + 4
// during emulation of any instruction using this implementation.
func (p *basicARMProcessor) RunNextInstruction() error {
	pc, e := p.GetRegisterNumber(15)
	if e != nil {
		return fmt.Errorf("Failed getting PC: %s", e)
	}
	if p.THUMBMode() {
		raw, e := p.GetMemoryInterface().ReadMemoryHalfword(pc)
		if e != nil {
			return fmt.Errorf("Failed fetching instruction: %s", e)
		}
		e = p.SetRegisterNumber(15, pc+2)
		if e != nil {
			return fmt.Errorf("Failed incrementing PC: %s", e)
		}
		instruction, e := ParseTHUMBInstruction(raw)
		if e != nil {
			return fmt.Errorf("Failed decoding 0x%04x: %s", raw, e)
		}
		return instruction.Emulate(p)
	}
	raw, e := p.GetMemoryInterface().ReadMemoryWord(pc)
	if e != nil {
		return fmt.Errorf("Failed fetching instruction: %s", e)
	}
	instruction, e := ParseInstruction(raw)
	if e != nil {
		return fmt.Errorf("Failed decoding 0x%08x: %s", raw, e)
	}
	e = p.SetRegisterNumber(15, pc+4)
	if e != nil {
		return fmt.Errorf("Failed incrementing PC: %s", e)
	}
	e = instruction.Emulate(p)
	if e != nil {
		return fmt.Errorf("Failed emulating instruction: %s", e)
	}
	return nil
}

func NewARMProcessor() ARMProcessor {
	var toReturn basicARMProcessor
	toReturn.memory = NewARMMemory()
	toReturn.currentStatusRegister = uint32(userMode)
	toReturn.coprocessors = make([]ARMCoprocessor, 0, 1)
	return &toReturn
}
