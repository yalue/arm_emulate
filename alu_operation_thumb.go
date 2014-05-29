package arm_emulate

import (
	"fmt"
)

var opcodeStringsTHUMB = [...]string{"and", "eor", "lsl", "lsr", "asr", "adc",
	"sbc", "ror", "tst", "neg", "cmp", "cmn", "orr", "mul", "bic", "mvn"}

// Similar to the ARMDataProcessingOpcode type, but for THUMB ALU instructions.
type ALUOpcodeTHUMB interface {
	fmt.Stringer
	Value() uint8
	// The THUMB equivalent of ARMDataProcessingOpcode's evaluate function.
	// Returns the result, whether the result should be stored and an error
	// if one occurred. Also sets processor flags.
	Evaluate(a uint32, b uint32, p ARMProcessor) (uint32, bool, error)
}

type basicALUOpcodeTHUMB struct {
	value uint8
}

func (o *basicALUOpcodeTHUMB) String() string {
	return opcodeStringsTHUMB[o.value&0xf]
}

func (o *basicALUOpcodeTHUMB) Value() uint8 {
	return o.value
}

// Returns the result of evalutationg the opcode and whether it should be
// stored. Also sets the carry and overflow flags.
func evaluateALUOperationTHUMB(a, b uint32, opcode uint8,
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
				// mul
				return a * b, true
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
			// neg
			return -b, true
		}
		// tst
		return a & b, false
	}
	if (opcode & 4) != 0 {
		if (opcode & 2) != 0 {
			if (opcode & 1) != 0 {
				// ror
				for b > 32 {
					b -= 32
				}
				if b == 0 {
					return a, true
				}
				a = (a << (32 - b)) | (32 >> b)
				p.SetCarry((a & 0x80000000) != 0)
				return a, true
			}
			// sbc
			carry := uint32(0)
			if p.Carry() {
				carry = 1
			}
			p.SetCarry(isCarryC(a, b, p.Carry(), true))
			p.SetOverflow(isOverflowC(a, b, p.Carry(), true))
			return a - b + carry - 1, true
		}
		if (opcode & 1) != 0 {
			// adc
			carry := uint32(0)
			if p.Carry() {
				carry = 1
				p.SetCarry(isCarryC(a, b, p.Carry(), false))
				p.SetOverflow(isOverflowC(a, b, p.Carry(), false))
				return a + b + carry, true
			}
		}
		// asr
		if b == 0 {
			return a, true
		}
		if b >= 32 {
			if (a & 0x80000000) != 0 {
				p.SetCarry(true)
				return 0xffffffff, true
			}
			p.SetCarry(false)
			return 0, true
		}
		p.SetCarry(((a >> (b - 1)) & 1) != 0)
		return uint32(int(a) >> b), true
	}
	if (opcode & 2) != 0 {
		if (opcode & 1) != 0 {
			// lsr
			if b == 0 {
				return a, true
			}
			if b > 32 {
				p.SetCarry(false)
				return 0, true
			}
			p.SetCarry(((a >> (b - 1)) & 1) != 0)
			return a >> b, true
		}
		// lsl
		if b == 0 {
			return a, true
		}
		if b > 32 {
			p.SetCarry(false)
			return 0, true
		}
		p.SetCarry(((a << (b - 1)) & 0x80000000) != 0)
		return a << b, true
	}
	if (opcode & 1) != 0 {
		// eor
		return a ^ b, true
	}
	// and
	return a & b, true
}

func (o *basicALUOpcodeTHUMB) Evaluate(a, b uint32, p ARMProcessor) (uint32,
	bool, error) {
	result, storeResult := evaluateALUOperationTHUMB(a, b, o.value, p)
	p.SetZero(result == 0)
	p.SetNegative((result & 0x80000000) != 0)
	return result, storeResult, nil
}

func NewALUOpcodeTHUMB(value uint8) ALUOpcodeTHUMB {
	var toReturn basicALUOpcodeTHUMB
	toReturn.value = value
	return &toReturn
}
