package arm_emulate

import (
	"fmt"
)

var shiftStrings = [...]string{"lsl", "lsr", "asr", "ror"}

// This interface is used when processing the shifts encoded in ARM data
// processing instructions.
type ARMShift interface {
	fmt.Stringer
	ShiftString() string
	Register() ARMRegister
	Amount() uint8
	UseRegister() bool
	ShiftType() uint8
	Apply(value uint32, p ARMProcessor) (uint32, error)
}

type basicARMShift struct {
	shiftType   uint8
	register    ARMRegister
	amount      uint8
	useRegister bool
}

func (s *basicARMShift) ShiftString() string {
	return shiftStrings[s.shiftType&3]
}

func (s *basicARMShift) String() string {
	shiftString := s.ShiftString()
	if s.useRegister {
		return fmt.Sprintf("%s %s", shiftString, s.register)
	}
	if s.amount == 0 {
		return ""
	}
	return fmt.Sprintf("%s %d", shiftString, s.amount)
}

func applyLSL(value uint32, amount uint8, registerSpecified bool,
	p ARMProcessor) (uint32, error) {
	if amount == 0 {
		return value, nil
	}
	if amount >= 32 {
		if amount == 32 {
			p.SetCarry((value & 1) != 0)
		}
		return 0, nil
	}
	p.SetCarry(((value << (amount - 1)) & 0x80000000) != 0)
	return value << amount, nil
}

func applyLSR(value uint32, amount uint8, registerSpecified bool,
	p ARMProcessor) (uint32, error) {
	if amount == 0 {
		if registerSpecified {
			return value, nil
		}
		// lsr <value>, 0 is used as lsr <value>, 32 instead
		p.SetCarry((value & 0x80000000) != 0)
		return 0, nil
	}
	if amount >= 32 {
		if amount == 32 {
			p.SetCarry((value & 0x80000000) != 0)
			return 0, nil
		}
		p.SetCarry(false)
		return 0, nil
	}
	p.SetCarry(((value >> (amount - 1)) & 1) != 0)
	return value >> amount, nil
}

func applyASR(value uint32, amount uint8, registerSpecified bool,
	p ARMProcessor) (uint32, error) {
	if amount == 0 {
		if registerSpecified {
			return value, nil
		}
		if (value & 0x80000000) != 0 {
			p.SetCarry(true)
			return 0xffffffff, nil
		}
		p.SetCarry(false)
		return 0, nil
	}
	if amount >= 32 {
		if (value & 0x80000000) != 0 {
			p.SetCarry(true)
			return 0xffffffff, nil
		}
		p.SetCarry(false)
		return 0, nil
	}
	p.SetCarry(((value >> (amount - 1)) & 1) != 0)
	return uint32(int32(value) >> amount), nil
}

func applyROR(value uint32, amount uint8, registerSpecified bool,
	p ARMProcessor) (uint32, error) {
	if amount == 0 {
		if registerSpecified {
			return value, nil
		}
		// ROR with immediate 0 instead is used as RRX
		toReturn := value >> 1
		if p.Carry() {
			toReturn |= 0x80000000
		}
		p.SetCarry((value & 1) != 0)
		return toReturn, nil
	}
	for amount > 32 {
		amount -= 32
	}
	if amount == 32 {
		p.SetCarry((value & 0x80000000) != 0)
		return value, nil
	}
	value = (value << (32 - amount)) | (value >> amount)
	p.SetCarry((value & 0x80000000) != 0)
	return value, nil
}

func (s *basicARMShift) Apply(value uint32, p ARMProcessor) (uint32, error) {
	var amount uint8
	if s.useRegister {
		if s.register.Register() == 15 {
			return value, fmt.Errorf("Can't use r15 in a shift.")
		}
		registerAmount, e := p.GetRegister(s.register)
		if e != nil {
			return value, e
		}
		amount = uint8(registerAmount & 0xff)
	} else {
		amount = s.amount
	}
	switch s.shiftType {
	case 0:
		return applyLSL(value, amount, s.useRegister, p)
	case 1:
		return applyLSR(value, amount, s.useRegister, p)
	case 2:
		return applyASR(value, amount, s.useRegister, p)
	case 3:
		return applyROR(value, amount, s.useRegister, p)
	}
	return value, fmt.Errorf("Invalid shift type: %d", s.shiftType)
}

func (s *basicARMShift) Register() ARMRegister {
	return s.register
}

func (s *basicARMShift) Amount() uint8 {
	return s.amount
}

func (s *basicARMShift) UseRegister() bool {
	return s.useRegister
}

func (s *basicARMShift) ShiftType() uint8 {
	return s.shiftType
}

func NewARMShift(shift uint8) ARMShift {
	var toReturn basicARMShift
	toReturn.useRegister = (shift & 1) == 1
	toReturn.shiftType = (shift >> 1) & 3
	if toReturn.useRegister {
		toReturn.register = NewARMRegister((shift >> 4) & 0xf)
	} else {
		toReturn.amount = (shift >> 3) & 0x1f
	}
	return &toReturn
}
