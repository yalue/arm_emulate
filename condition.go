package arm_emulate

var conditionStrings = [...]string{"eq", "ne", "cs", "cc", "mi", "pl", "vs",
	"vc", "hi", "ls", "ge", "lt", "gt", "le", "al", "nv"}

type ARMCondition uint8

func (c ARMCondition) String() string {
	if c == 14 {
		return ""
	}
	return conditionStrings[c&0xf]
}

func (c ARMCondition) IsMet(p ARMProcessor) bool {
	switch c {
	case 14:
		return true
	case 0:
		return p.Zero()
	case 1:
		return !p.Zero()
	case 2:
		return p.Carry()
	case 3:
		return !p.Carry()
	case 4:
		return p.Negative()
	case 5:
		return !p.Negative()
	case 6:
		return p.Overflow()
	case 7:
		return !p.Overflow()
	case 8:
		return p.Carry() && !p.Zero()
	case 9:
		return !p.Carry() || p.Zero()
	case 10:
		return p.Negative() == p.Overflow()
	case 11:
		return p.Negative() != p.Overflow()
	case 12:
		return p.Zero() && (p.Negative() == p.Overflow())
	case 13:
		return !p.Zero() || (p.Negative() != p.Overflow())
	}
	// Case 15-- error?
	return false
}
