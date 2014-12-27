package arm_emulate

import (
	"fmt"
)

func (n *MoveShiftedRegisterInstruction) Emulate(p ARMProcessor) error {
	value, _ := p.GetRegister(n.Rs)
	var result uint32
	switch n.Operation {
	case 0:
		// Logical shift left
		if n.Offset == 0 {
			result = value
			break
		}
		p.SetCarry(((value << (n.Offset - 1)) & 0x80000000) != 0)
		result = value << n.Offset
	case 1:
		// Logical shift right
		if n.Offset == 0 {
			p.SetCarry((value & 0x80000000) != 0)
			result = 0
			break
		}
		p.SetCarry(((value >> (n.Offset - 1)) & 1) != 0)
		result = value >> n.Offset
	case 2:
		// Arithmetic shift right
		if n.Offset == 0 {
			if (value & 0x80000000) != 0 {
				p.SetCarry(true)
				result = 0xffffffff
			} else {
				p.SetCarry(false)
				result = 0
			}
			break
		}
		p.SetCarry(((value >> (n.Offset - 1)) & 1) != 0)
		result = uint32(int32(value) >> n.Offset)
	default:
		return fmt.Errorf("Invalid shift operation.")
	}
	p.SetZero(result == 0)
	p.SetNegative((result & 0x80000000) != 0)
	p.SetRegister(n.Rd, result)
	return nil
}

func (n *AddSubtractInstruction) Emulate(p ARMProcessor) error {
	start, _ := p.GetRegister(n.Rs)
	var difference uint32
	if n.IsImmediate {
		difference = uint32(n.Immediate)
	} else {
		difference, _ = p.GetRegister(n.Rn)
	}
	var result uint32
	if n.Subtract {
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
	p.SetRegister(n.Rd, result)
	return nil
}

func (n *MoveCompareAddSubtractImmediateInstruction) Emulate(
	p ARMProcessor) error {
	var newValue uint32
	startValue, _ := p.GetRegister(n.Rd)
	difference := uint32(n.Immediate)
	switch n.Operation {
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
		return fmt.Errorf("Invalid operation: %d", n.Operation)
	}
	p.SetZero(newValue == 0)
	p.SetNegative((newValue & 0x80000000) != 0)
	if n.Operation != 1 {
		p.SetRegister(n.Rd, newValue)
	}
	return nil
}

func (n *ALUOperationInstruction) Emulate(p ARMProcessor) error {
	a, _ := p.GetRegister(n.Rd)
	b, _ := p.GetRegister(n.Rs)
	result, storeResult, e := n.Opcode.Evaluate(a, b, p)
	if e != nil {
		return fmt.Errorf("ALU operation failed: %s", e)
	}
	if storeResult {
		p.SetRegister(n.Rd, result)
	}
	return nil
}

func (n *HighRegisterOperationInstruction) Emulate(p ARMProcessor) error {
	a, _ := p.GetRegister(n.Rd)
	if n.Rd == 15 {
		a += 2
	}
	b, _ := p.GetRegister(n.Rs)
	if n.Rs == 15 {
		b += 2
	}
	switch n.Operation {
	case 0:
		// add
		p.SetRegister(n.Rd, a+b)
	case 1:
		// cmp
		p.SetCarry(isCarry(a, b, true))
		p.SetOverflow(isOverflow(a, b, true))
		result := a - b
		p.SetZero(result == 0)
		p.SetNegative((result & 0x80000000) != 0)
	case 2:
		p.SetRegister(n.Rd, b)
	case 3:
		if (b & 1) == 0 {
			e := p.SetTHUMBMode(false)
			if e != nil {
				return e
			}
		}
		p.SetRegister(15, b)
	}
	return nil
}

func (n *PcRelativeLoadInstruction) Emulate(p ARMProcessor) error {
	base, _ := p.GetRegister(15)
	base += 2
	base &= 0xfffffffc
	base += uint32(n.Offset) << 2
	value, e := p.GetMemoryInterface().ReadMemoryWord(base)
	if e != nil {
		return e
	}
	p.SetRegister(n.Rd, value)
	return nil
}

func (n *LoadStoreRegisterOffsetInstruction) Emulate(p ARMProcessor) error {
	base, _ := p.GetRegister(n.Rb)
	offset, _ := p.GetRegister(n.Ro)
	base += offset
	var e error
	m := p.GetMemoryInterface()
	if n.Load {
		var loaded uint32
		var b uint8
		if n.ByteQuantity {
			b, e = m.ReadMemoryByte(base)
			loaded = uint32(b)
		} else {
			loaded, e = m.ReadMemoryWord(base)
		}
		if e != nil {
			return e
		}
		p.SetRegister(n.Rd, loaded)
	} else {
		toStore, _ := p.GetRegister(n.Rd)
		if n.ByteQuantity {
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

func (n *LoadStoreSignExtendedHalfwordInstruction) Emulate(
	p ARMProcessor) error {
	address, _ := p.GetRegister(n.Rb)
	offset, _ := p.GetRegister(n.Ro)
	address += offset
	m := p.GetMemoryInterface()
	if n.SignExtend {
		var extended uint32
		if n.HBit {
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
		p.SetRegister(n.Rd, extended)
		return nil
	}
	if n.HBit {
		loaded, e := m.ReadMemoryHalfword(address)
		if e != nil {
			return e
		}
		p.SetRegister(n.Rd, uint32(loaded))
		return nil
	}
	toStore, _ := p.GetRegister(n.Rd)
	e := m.WriteMemoryHalfword(address, uint16(toStore))
	return e
}

func (n *LoadStoreImmediateOffsetInstruction) Emulate(p ARMProcessor) error {
	address, _ := p.GetRegister(n.Rb)
	m := p.GetMemoryInterface()
	if n.ByteQuantity {
		address += uint32(n.Offset)

	} else {
		address += uint32(n.Offset) << 2
	}
	if n.Load {
		var loaded uint32
		var e error
		var byteValue uint8
		if n.ByteQuantity {
			byteValue, e = m.ReadMemoryByte(address)
			loaded = uint32(byteValue)
		} else {
			loaded, e = m.ReadMemoryWord(address)
		}
		if e != nil {
			return e
		}
		p.SetRegister(n.Rd, loaded)
		return nil
	}
	toStore, _ := p.GetRegister(n.Rd)
	if n.ByteQuantity {
		return m.WriteMemoryByte(address, uint8(toStore))
	}
	return m.WriteMemoryWord(address, toStore)
}

func (n *LoadStoreHalfwordInstruction) Emulate(p ARMProcessor) error {
	address, _ := p.GetRegister(n.Rb)
	address += uint32(n.Offset) << 1
	m := p.GetMemoryInterface()
	if n.Load {
		loaded, e := m.ReadMemoryHalfword(address)
		if e != nil {
			return e
		}
		return p.SetRegister(n.Rd, uint32(loaded))
	}
	toWrite, _ := p.GetRegister(n.Rd)
	return m.WriteMemoryHalfword(address, uint16(toWrite))
}

func (n *SPRelativeLoadStoreInstruction) Emulate(p ARMProcessor) error {
	address, _ := p.GetRegister(13)
	address += uint32(n.Offset) << 2
	m := p.GetMemoryInterface()
	if n.Load {
		value, e := m.ReadMemoryWord(address)
		if e != nil {
			return e
		}
		return p.SetRegister(n.Rd, value)
	}
	value, _ := p.GetRegister(n.Rd)
	return m.WriteMemoryWord(address, value)
}

func (n *LoadAddressInstruction) Emulate(p ARMProcessor) error {
	var value uint32
	if n.LoadSP {
		value, _ = p.GetRegister(13)
	} else {
		value, _ = p.GetRegister(15)
		value += 2
		value &= 0xfffffffc
	}
	value += uint32(n.Offset) << 2
	return p.SetRegister(n.Rd, value)
}

func (n *AddToStackPointerInstruction) Emulate(p ARMProcessor) error {
	start, _ := p.GetRegister(13)
	offset := uint32(n.Offset) << 2
	if n.Negative {
		start -= offset
	} else {
		start += offset
	}
	return p.SetRegister(13, start)
}

func (n *PushPopRegistersInstruction) pushRegisters(p ARMProcessor) error {
	baseAddress, _ := p.GetRegister(13)
	bits := n.RegisterList
	toStore := make([]uint32, 0, 9)
	for i := 0; (i < 8) && (bits != 0); i++ {
		if (bits & 1) == 1 {
			registerContents, e := p.GetRegister(ARMRegister(i))
			if e != nil {
				return e
			}
			toStore = append(toStore, registerContents)
		}
		bits = bits >> 1
	}
	if n.StoreLRLoadPC {
		lrContents, e := p.GetRegister(14)
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
	p.SetRegister(13, baseAddress)
	return nil
}

func (n *PushPopRegistersInstruction) popRegisters(p ARMProcessor) error {
	baseAddress, _ := p.GetRegister(13)
	bits := n.RegisterList
	toLoad := make([]uint8, 0, 9)
	for i := 0; (i < 8) && (bits != 0); i++ {
		if (bits & 1) == 1 {
			toLoad = append(toLoad, uint8(i))
		}
		bits = bits >> 1
	}
	if n.StoreLRLoadPC {
		toLoad = append(toLoad, 15)
	}
	m := p.GetMemoryInterface()
	for _, registerNumber := range toLoad {
		value, e := m.ReadMemoryWord(baseAddress)
		if e != nil {
			return e
		}
		e = p.SetRegister(ARMRegister(registerNumber), value)
		if e != nil {
			return e
		}
		baseAddress += 4
	}
	return nil
}

func (n *PushPopRegistersInstruction) Emulate(p ARMProcessor) error {
	if n.Load {
		return n.popRegisters(p)
	}
	return n.pushRegisters(p)
}

func (n *MultipleLoadStoreInstruction) multipleStoreTHUMB(
	p ARMProcessor) error {
	baseAddress, _ := p.GetRegister(n.Rb)
	bits := n.RegisterList
	toStore := make([]uint32, 0, 8)
	for i := 0; (i < 8) && (bits != 0); i++ {
		if (bits & 1) == 1 {
			registerContents, e := p.GetRegister(ARMRegister(i))
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
	return p.SetRegister(n.Rb, baseAddress)
}

func (n *MultipleLoadStoreInstruction) multipleLoadTHUMB(
	p ARMProcessor) error {
	baseAddress, _ := p.GetRegister(n.Rb)
	bits := n.RegisterList
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
		e = p.SetRegister(ARMRegister(registerNumber), value)
		if e != nil {
			return e
		}
		baseAddress += 4
	}
	return p.SetRegister(n.Rb, baseAddress)
}

func (n *MultipleLoadStoreInstruction) Emulate(p ARMProcessor) error {
	if n.Load {
		return n.multipleLoadTHUMB(p)
	}
	return n.multipleStoreTHUMB(p)
}

func (n *ConditionalBranchInstruction) Emulate(p ARMProcessor) error {
	if !n.Condition.IsMet(p) {
		return nil
	}
	offset := (int32(n.Offset) << 24) >> 23
	address, _ := p.GetRegister(15)
	address += 2
	address = uint32(int32(address) + offset)
	return p.SetRegister(15, address)
}

func (n *SoftwareInterruptTHUMBInstruction) Emulate(p ARMProcessor) error {
	currentPC, _ := p.GetRegister(15)
	e := p.SetMode(0x13)
	if e != nil {
		return e
	}
	p.SetRegister(14, currentPC)
	p.SetRegister(15, 0x8)
	return p.SetTHUMBMode(false)
}

func (n *UnconditionalBranchInstruction) Emulate(p ARMProcessor) error {
	offset := (int32(n.Offset) << 21) >> 20
	current, _ := p.GetRegister(15)
	current += 2
	target := uint32(int32(current) + offset)
	return p.SetRegister(15, target)
}

func (n *LongBranchAndLinkInstruction) Emulate(p ARMProcessor) error {
	currentPC, _ := p.GetRegister(15)
	if n.OffsetLow {
		currentLR, _ := p.GetRegister(14)
		currentLR += uint32(n.Offset) << 1
		p.SetRegister(14, currentPC|1)
		p.SetRegister(15, currentLR)
		return nil
	}
	currentPC += 2
	currentPC += uint32((int32(n.Offset) << 21) >> 9)
	p.SetRegister(14, currentPC)
	return nil
}
