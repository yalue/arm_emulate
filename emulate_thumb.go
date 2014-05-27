package arm_emulate

import (
	"fmt"
)

func (n *moveShiftedRegisterInstruction) Emulate(p ARMProcessor) error {
	value, _ := p.GetRegister(n.rs)
	var result uint32
	switch n.operation {
	case 0:
		// Logical shift left
		if n.offset == 0 {
			result = value
			break
		}
		p.SetCarry(((value << (n.offset - 1)) & 0x80000000) != 0)
		result = value << n.offset
	case 1:
		// Logical shift right
		if n.offset == 0 {
			p.SetCarry((value & 0x80000000) != 0)
			result = 0
			break
		}
		p.SetCarry(((value >> (n.offset - 1)) & 1) != 0)
		result = value >> n.offset
	case 2:
		// Arithmetic shift right
		if n.offset == 0 {
			if (value & 0x80000000) != 0 {
				p.SetCarry(true)
				result = 0xffffffff
			} else {
				p.SetCarry(false)
				result = 0
			}
			break
		}
		p.SetCarry(((value >> (n.offset - 1)) & 1) != 0)
		result = uint32(int32(value) >> n.offset)
	default:
		return fmt.Errorf("Invalid shift operation.")
	}
	p.SetZero(result == 0)
	p.SetNegative((result & 0x80000000) != 0)
	p.SetRegister(n.rd, result)
	return nil
}

func (n *addSubtractInstruction) Emulate(p ARMProcessor) error {
	start, _ := p.GetRegister(n.rs)
	var difference uint32
	if n.isImmediate {
		difference = uint32(n.immediate)
	} else {
		difference, _ = p.GetRegister(n.rn)
	}
	var result uint32
	if n.subtract {
		p.SetCarry(isCarry(start, difference, true))
		p.SetOverflow(isOverflow(start, difference, true))
		result = start - difference
	} else {
		p.SetCarry(isCarry(start, difference, false))
		p.SetOverflow(isOverflow(start, difference, false))
		result = start + difference
	}
	p.SetZero(result == 0)
	p.SetNegative((result & 0x80000000) != 0)
	p.SetRegister(n.rd, result)
	return nil
}

func (n *moveCompareAddSubtractImmediateInstruction) Emulate(
	p ARMProcessor) error {
	var newValue uint32
	startValue, _ := p.GetRegister(n.rd)
	difference := uint32(n.immediate)
	switch n.operation {
	case 0:
		// mov
		newValue = difference
	case 1:
		// cmp
		p.SetCarry(isCarry(startValue, difference, true))
		p.SetOverflow(isOverflow(startValue, difference, true))
		newValue = startValue - difference
	case 2:
		// add
		p.SetCarry(isCarry(startValue, difference, false))
		p.SetOverflow(isOverflow(startValue, difference, false))
		newValue = startValue + difference
	case 3:
		// sub
		p.SetCarry(isCarry(startValue, difference, true))
		p.SetOverflow(isOverflow(startValue, difference, true))
		newValue = startValue - difference
	default:
		return fmt.Errorf("Invalid operation: %d", n.operation)
	}
	p.SetZero(newValue == 0)
	p.SetNegative((newValue & 0x80000000) != 0)
	if n.operation != 1 {
		p.SetRegister(n.rd, newValue)
	}
	return nil
}

func (n *aluOperationInstruction) Emulate(p ARMProcessor) error {
	a, _ := p.GetRegister(n.rd)
	b, _ := p.GetRegister(n.rs)
	result, storeResult, e := n.opcode.Evaluate(a, b, p)
	if e != nil {
		return fmt.Errorf("ALU operation failed: %s", e)
	}
	if storeResult {
		p.SetRegister(n.rd, result)
	}
	return nil
}

func (n *highRegisterOperationInstruction) Emulate(p ARMProcessor) error {
	a, _ := p.GetRegister(n.rd)
	if n.rd.Register() == 15 {
		a += 2
	}
	b, _ := p.GetRegister(n.rs)
	if n.rs.Register() == 15 {
		b += 2
	}
	switch n.operation {
	case 0:
		// add
		p.SetRegister(n.rd, a+b)
	case 1:
		// cmp
		p.SetCarry(isCarry(a, b, true))
		p.SetOverflow(isOverflow(a, b, true))
		result := a - b
		p.SetZero(result == 0)
		p.SetNegative((result & 0x80000000) != 0)
	case 2:
		p.SetRegister(n.rd, b)
	case 3:
		if (b & 1) == 0 {
			e := p.SetTHUMBMode(false)
			if e != nil {
				return e
			}
		}
		p.SetRegisterNumber(15, b)
	}
	return nil
}

func (n *pcRelativeLoadInstruction) Emulate(p ARMProcessor) error {
	base, _ := p.GetRegisterNumber(15)
	base += 2
	base &= 0xfffffffc
	base += uint32(n.offset) << 2
	value, e := p.GetMemoryInterface().ReadMemoryWord(base)
	if e != nil {
		return e
	}
	p.SetRegister(n.rd, value)
	return nil
}

func (n *loadStoreRegisterOffsetInstruction) Emulate(p ARMProcessor) error {
	base, _ := p.GetRegister(n.rb)
	offset, _ := p.GetRegister(n.ro)
	base += offset
	var e error
	m := p.GetMemoryInterface()
	if n.load {
		var loaded uint32
		var b uint8
		if n.byteQuantity {
			b, e = m.ReadMemoryByte(base)
			loaded = uint32(b)
		} else {
			loaded, e = m.ReadMemoryWord(base)
		}
		if e != nil {
			return e
		}
		p.SetRegister(n.rd, loaded)
	} else {
		toStore, _ := p.GetRegister(n.rd)
		if n.byteQuantity {
			e = m.WriteMemoryByte(base, uint8(toStore))
		} else {
			e = m.WriteMemoryWord(base, toStore)
		}
		if e != nil {
			return e
		}
	}
	return nil
}

func (n *loadStoreSignExtendedHalfwordInstruction) Emulate(
	p ARMProcessor) error {
	address, _ := p.GetRegister(n.rb)
	offset, _ := p.GetRegister(n.ro)
	address += offset
	m := p.GetMemoryInterface()
	if n.signExtend {
		var extended uint32
		if n.hBit {
			loaded, e := m.ReadMemoryHalfword(address)
			if e != nil {
				return e
			}
			extended = uint32((int32(loaded) << 16) >> 16)
		} else {
			loaded, e := m.ReadMemoryByte(address)
			if e != nil {
				return e
			}
			extended = uint32((int32(loaded) << 24) >> 24)
		}
		p.SetRegister(n.rd, extended)
		return nil
	}
	if n.hBit {
		loaded, e := m.ReadMemoryHalfword(address)
		if e != nil {
			return e
		}
		p.SetRegister(n.rd, uint32(loaded))
		return nil
	}
	toStore, _ := p.GetRegister(n.rd)
	e := m.WriteMemoryHalfword(address, uint16(toStore))
	return e
}

func (n *loadStoreImmediateOffsetInstruction) Emulate(p ARMProcessor) error {
	address, _ := p.GetRegister(n.rb)
	m := p.GetMemoryInterface()
	if n.byteQuantity {
		address += uint32(n.offset)

	} else {
		address += uint32(n.offset) << 2
	}
	if n.load {
		var loaded uint32
		var e error
		var byteValue uint8
		if n.byteQuantity {
			byteValue, e = m.ReadMemoryByte(address)
			loaded = uint32(byteValue)
		} else {
			loaded, e = m.ReadMemoryWord(address)
		}
		if e != nil {
			return e
		}
		p.SetRegister(n.rd, loaded)
		return nil
	}
	toStore, _ := p.GetRegister(n.rd)
	if n.byteQuantity {
		return m.WriteMemoryByte(address, uint8(toStore))
	}
	return m.WriteMemoryWord(address, toStore)
}

func (n *loadStoreHalfwordInstruction) Emulate(p ARMProcessor) error {
	address, _ := p.GetRegister(n.rb)
	address += uint32(n.offset) << 1
	m := p.GetMemoryInterface()
	if n.load {
		loaded, e := m.ReadMemoryHalfword(address)
		if e != nil {
			return e
		}
		return p.SetRegister(n.rd, uint32(loaded))
	}
	toWrite, _ := p.GetRegister(n.rd)
	return m.WriteMemoryHalfword(address, uint16(toWrite))
}

func (n *spRelativeLoadStoreInstruction) Emulate(p ARMProcessor) error {
	address, _ := p.GetRegisterNumber(13)
	address += uint32(n.offset) << 2
	m := p.GetMemoryInterface()
	if n.load {
		value, e := m.ReadMemoryWord(address)
		if e != nil {
			return e
		}
		return p.SetRegister(n.rd, value)
	}
	value, _ := p.GetRegister(n.rd)
	return m.WriteMemoryWord(address, value)
}

func (n *loadAddressInstruction) Emulate(p ARMProcessor) error {
	var value uint32
	if n.loadSP {
		value, _ = p.GetRegisterNumber(13)
	} else {
		value, _ = p.GetRegisterNumber(15)
		value += 2
		value &= 0xfffffffc
	}
	value += uint32(n.offset) << 2
	return p.SetRegister(n.rd, value)
}

func (n *addToStackPointerInstruction) Emulate(p ARMProcessor) error {
	start, _ := p.GetRegisterNumber(13)
	offset := uint32(n.offset) << 2
	if n.negative {
		start -= offset
	} else {
		start += offset
	}
	return p.SetRegisterNumber(13, start)
}

func (n *pushPopRegistersInstruction) pushRegisters(p ARMProcessor) error {
	baseAddress, _ := p.GetRegisterNumber(13)
	bits := n.registerList
	toStore := make([]uint32, 0, 9)
	for i := 0; (i < 8) && (bits != 0); i++ {
		if (bits & 1) == 1 {
			registerContents, e := p.GetRegisterNumber(uint8(i))
			if e != nil {
				return e
			}
			toStore = append(toStore, registerContents)
		}
		bits = bits >> 1
	}
	if n.storeLRLoadPC {
		lrContents, e := p.GetRegisterNumber(14)
		if e != nil {
			return e
		}
		toStore = append(toStore, lrContents)
	}
	// Reverse the order of values to store...
	for i, j := 0, len(toStore)-1; i < j; i, j = i+1, j-1 {
		toStore[i], toStore[j] = toStore[j], toStore[i]
	}
	m := p.GetMemoryInterface()
	for _, value := range toStore {
		baseAddress -= 4
		e := m.WriteMemoryWord(baseAddress, value)
		if e != nil {
			return e
		}
	}
	p.SetRegisterNumber(13, baseAddress)
	return nil
}

func (n *pushPopRegistersInstruction) popRegisters(p ARMProcessor) error {
	baseAddress, _ := p.GetRegisterNumber(13)
	bits := n.registerList
	toLoad := make([]uint8, 0, 9)
	for i := 0; (i < 8) && (bits != 0); i++ {
		if (bits & 1) == 1 {
			toLoad = append(toLoad, uint8(i))
		}
		bits = bits >> 1
	}
	if n.storeLRLoadPC {
		toLoad = append(toLoad, 15)
	}
	m := p.GetMemoryInterface()
	for _, registerNumber := range toLoad {
		value, e := m.ReadMemoryWord(baseAddress)
		if e != nil {
			return e
		}
		e = p.SetRegisterNumber(registerNumber, value)
		if e != nil {
			return e
		}
		baseAddress += 4
	}
	return nil
}

func (n *pushPopRegistersInstruction) Emulate(p ARMProcessor) error {
	if n.load {
		return n.popRegisters(p)
	}
	return n.pushRegisters(p)
}

func (n *multipleLoadStoreInstruction) multipleStoreTHUMB(
	p ARMProcessor) error {
	baseAddress, _ := p.GetRegister(n.rb)
	bits := n.registerList
	toStore := make([]uint32, 0, 8)
	for i := 0; (i < 8) && (bits != 0); i++ {
		if (bits & 1) == 1 {
			registerContents, e := p.GetRegisterNumber(uint8(i))
			if e != nil {
				return e
			}
			toStore = append(toStore, registerContents)
		}
		bits = bits >> 1
	}
	m := p.GetMemoryInterface()
	for _, value := range toStore {
		e := m.WriteMemoryWord(baseAddress, value)
		if e != nil {
			return e
		}
		baseAddress += 4
	}
	return p.SetRegister(n.rb, baseAddress)
}

func (n *multipleLoadStoreInstruction) multipleLoadTHUMB(
	p ARMProcessor) error {
	baseAddress, _ := p.GetRegister(n.rb)
	bits := n.registerList
	toLoad := make([]uint8, 0, 8)
	for i := 0; (i < 8) && (bits != 0); i++ {
		if (bits & 1) == 1 {
			toLoad = append(toLoad, uint8(i))
		}
		bits = bits >> 1
	}
	m := p.GetMemoryInterface()
	for _, registerNumber := range toLoad {
		value, e := m.ReadMemoryWord(baseAddress)
		if e != nil {
			return e
		}
		e = p.SetRegisterNumber(registerNumber, value)
		if e != nil {
			return e
		}
		baseAddress += 4
	}
	return p.SetRegister(n.rb, baseAddress)
}

func (n *multipleLoadStoreInstruction) Emulate(p ARMProcessor) error {
	if n.load {
		return n.multipleLoadTHUMB(p)
	}
	return n.multipleStoreTHUMB(p)
}

func (n *conditionalBranchInstruction) Emulate(p ARMProcessor) error {
	if !n.condition.IsMet(p) {
		return nil
	}
	offset := (int32(n.offset) << 24) >> 23
	address, _ := p.GetRegisterNumber(15)
	address += 2
	address = uint32(int32(address) + offset)
	return p.SetRegisterNumber(15, address)
}

func (n *softwareInterruptTHUMBInstruction) Emulate(p ARMProcessor) error {
	currentPC, _ := p.GetRegisterNumber(15)
	e := p.SetMode(0x13)
	if e != nil {
		return e
	}
	p.SetRegisterNumber(14, currentPC)
	p.SetRegisterNumber(15, 0x8)
	return p.SetTHUMBMode(false)
}

func (n *unconditionalBranchInstruction) Emulate(p ARMProcessor) error {
	offset := (int32(n.offset) << 21) >> 20
	current, _ := p.GetRegisterNumber(15)
	current += 2
	target := uint32(int32(current) + offset)
	return p.SetRegisterNumber(15, target)
}

func (n *longBranchAndLinkInstruction) Emulate(p ARMProcessor) error {
	currentPC, _ := p.GetRegisterNumber(15)
	if n.offsetLow {
		currentLR, _ := p.GetRegisterNumber(14)
		currentLR += uint32(n.offset) << 1
		p.SetRegisterNumber(14, currentPC|1)
		p.SetRegisterNumber(15, currentLR)
		return nil
	}
	currentPC += 2
	currentPC += uint32((int32(n.offset) << 21) >> 9)
	p.SetRegisterNumber(14, currentPC)
	return nil
}
