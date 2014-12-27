package arm_emulate

const (
	andARMOpcode ARMDataProcessingOpcode = iota
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

type ARMDataProcessingOpcode uint8

// Returns the result of evaluating the opcode, where b is the second operand
// and a is the first. Also returns whether the result should be stored. If the
// operation is arithmetic, this will also set the carry and overflow flags.
func evaluateOperation(a uint32, b uint32, opcode ARMDataProcessingOpcode,
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

// This evaluates the operation and sets the processor condition flags.
// The returned boolean will be false if the returned result shouldn't be
// stored.
func (o ARMDataProcessingOpcode) Evaluate(a, b uint32,
	p ARMProcessor) (uint32, bool, error) {
	signMask := uint32(0x80000000)
	result, storeResult := evaluateOperation(a, b, o, p)
	p.SetZero(result == 0)
	p.SetNegative((result & signMask) != 0)
	return result, storeResult, nil
}

func (o ARMDataProcessingOpcode) String() string {
	return opcodeStrings[o&0xf]
}
