package arm_emulate

// Returns the overflow flag for the two inputs
func isOverflow(a, b uint32, sub bool) bool {
	aSign := (a & 0x80000000) != 0
	bSign := (b & 0x80000000) != 0
	if sub {
		resultSign := ((a - b) & 0x80000000) != 0
		return (aSign != bSign) && (resultSign == bSign)
	}
	resultSign := ((a + b) & 0x80000000) != 0
	return (aSign == bSign) && (resultSign != aSign)
}

// Returns the carry flag for the two inputs
func isCarry(a, b uint32, sub bool) bool {
	if sub {
		result := a - b
		return result > a
	}
	result := a + b
	if a > b {
		return result < a
	}
	return result < b
}

// Same is isCarry, but takes the carryIn flag as well
func isCarryC(a, b uint32, carryIn, sub bool) bool {
	carryValue := uint32(0)
	if carryIn {
		carryValue = 1
	}
	if sub {
		result := a - b + (carryValue - 1)
		return result > a
	}
	result := a + b + carryValue
	if a > b {
		return result < a
	}
	return result < b
}

// Same as isOverflow, but takes the carryIn flag as well
func isOverflowC(a, b uint32, carryIn, sub bool) bool {
	aSign := (a & 0x80000000) != 0
	bSign := (b & 0x80000000) != 0
	carryValue := uint32(0)
	if carryIn {
		carryValue = 1
	}
	if sub {
		resultSign := ((a - b + carryValue - 1) & 0x80000000) != 0
		return (aSign != bSign) && (resultSign == bSign)
	}
	resultSign := ((a + b + carryValue) & 0x80000000) != 0
	return (aSign == bSign) && (resultSign != aSign)
}
