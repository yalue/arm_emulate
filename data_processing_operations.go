package arm_emulate

import (
	"fmt"
)

const (
	andARMOpcode uint8 = iota
	eorARMOpcode
	subARMOpcode
	rsbARMOpcode
	addARMOpcode
	adcARMOpcode
	sbcARMOpcode
	rscARMOpcode
	tstARMOpcode
	teqARMOpcode
	cmpARMOpcode
	cmnARMOpcode
	orrARMOpcode
	movARMOpcode
	bicARMOpcode
	mvnARMOpcode
)

var opcodeStrings = [...]string{"and", "eor", "sub", "rsb", "add", "adc",
	"sbc", "rsc", "tst", "teq", "cmp", "cmn", "orr", "mov", "bic", "mvn"}

// This type is used for storing and evaluating the different opcodes used by
// ARM data processing instructions.
type ARMDataProcessingOpcode interface {
	fmt.Stringer
	Value() uint8
	// This evaluates the operation and sets the processor condition flags.
	// The returned boolean will be false if the returned result shouldn't be
	// stored.
	Evaluate(a, b uint32, p ARMProcessor) (uint32, bool, error)
}

type basicARMDataProcessingOpcode struct {
	value uint8
}

// Returns the result of evaluating the opcode, where b is the second operand
// and a is the first. Also returns whether the result should be stored. If the
// operation is arithmetic, this will also set the carry and overflow flags.
func evaluateOperation(a uint32, b uint32, opcode uint8,
	p ARMProcessor) (uint32, bool) {
	if (opcode & 8) != 0 {
		if (opcode & 4) != 0 {
			if (opcode & 2) != 0 {
				if (opcode & 1) != 0 {
					// mvn
					return ^b, true
				}
				// bic
				return a &^ b, true
			}
			if (opcode & 1) != 0 {
				// mov
				return b, true
			}
			// orr
			return a | b, true
		}
		if (opcode & 2) != 0 {
			if (opcode & 1) != 0 {
				// cmn
				p.SetCarry(isCarry(a, b, false))
				p.SetOverflow(isOverflow(a, b, false))
				return a + b, false
			}
			// cmp
			p.SetCarry(isCarry(a, b, true))
			p.SetOverflow(isOverflow(a, b, true))
			return a - b, false
		}
		if (opcode & 1) != 0 {
			// teq
			return a ^ b, false
		}
		// tst
		return a & b, false
	}
	if (opcode & 4) != 0 {
		carry := uint32(0)
		if p.Carry() {
			carry = 1
		}
		if (opcode & 2) != 0 {
			if (opcode & 1) != 0 {
				// rsc
				p.SetCarry(isCarryC(b, a, p.Carry(), true))
				p.SetOverflow(isOverflowC(b, a, p.Carry(), true))
				return b - a + carry - 1, true
			}
			// sbc
			p.SetCarry(isCarryC(a, b, p.Carry(), true))
			p.SetOverflow(isOverflowC(a, b, p.Carry(), true))
			return a - b + carry - 1, true
		}
		if (opcode & 1) != 0 {
			// adc
			p.SetCarry(isCarryC(a, b, p.Carry(), false))
			p.SetOverflow(isOverflowC(a, b, p.Carry(), false))
			return a + b + carry, true
		}
		// add
		p.SetCarry(isCarry(a, b, false))
		p.SetOverflow(isOverflow(a, b, false))
		return a + b, true
	}
	if (opcode & 2) != 0 {
		if (opcode & 1) != 0 {
			// rsb
			p.SetCarry(isCarry(b, a, true))
			p.SetOverflow(isOverflow(b, a, true))
			return b - a, true
		}
		// sub
		p.SetCarry(isCarry(a, b, true))
		p.SetOverflow(isOverflow(a, b, true))
		return a - b, true
	}
	if (opcode & 1) != 0 {
		// eor
		return a ^ b, true
	}
	// and
	return a & b, true
}

func (o *basicARMDataProcessingOpcode) Evaluate(a, b uint32,
	p ARMProcessor) (uint32, bool, error) {
	signMask := uint32(0x80000000)
	result, storeResult := evaluateOperation(a, b, o.value, p)
	p.SetZero(result == 0)
	p.SetNegative((result & signMask) != 0)
	return result, storeResult, nil
}

func (o *basicARMDataProcessingOpcode) String() string {
	return opcodeStrings[o.value&0xf]
}

func (o *basicARMDataProcessingOpcode) Value() uint8 {
	return o.value
}

func NewARMDataProcessingOpcode(value uint8) ARMDataProcessingOpcode {
	var toReturn basicARMDataProcessingOpcode
	toReturn.value = value & 0xf
	return &toReturn
}
