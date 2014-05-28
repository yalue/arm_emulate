package arm_emulate

import (
	"fmt"
)

func (n *DataProcessingInstruction) evaluateSecondOperand(
	p ARMProcessor) (uint32, error) {
	if n.isImmediate {
		r := n.rotate << 1
		value := uint32(n.immediate)
		return (value >> r) | (value << (32 - r)), nil
	}
	value, _ := p.GetRegister(n.rm)
	if n.rm.Register() == 15 {
		value += 4
		if n.shift.UseRegister() {
			value += 4
		}
	}
	return n.shift.Apply(value, p)
}

func (n *DataProcessingInstruction) Emulate(p ARMProcessor) error {
	if !n.Condition().IsMet(p) {
		return nil
	}
	// We let the opcode handlers set conditions always, and we'll restore them
	// if the 's' bit is clear.
	previousConditions, e := p.GetCPSR()
	if e != nil {
		return e
	}
	operand2, e := n.evaluateSecondOperand(p)
	if e != nil {
		return fmt.Errorf("Invalid second operand: %s", e)
	}
	operand1, _ := p.GetRegister(n.rn)
	if n.rn.Register() == 15 {
		operand1 += 4
		if !n.isImmediate && n.shift.UseRegister() {
			operand1 += 4
		}
	}
	result, writeResult, e := n.opcode.Evaluate(operand1, operand2, p)
	if e != nil {
		return e
	}
	if writeResult {
		p.SetRegister(n.rd, result)
		// Writes to r15 with the S bit set is an atomic operation to switch
		// modes (not valid in user mode)
		if (n.rd.Register() == 15) && n.setConditions {
			savedStatus, e := p.GetSPSR()
			if e != nil {
				return fmt.Errorf("Invalid write to r15 in user mode: %s", e)
			}
			e = p.SetCPSR(savedStatus)
			if e != nil {
				return fmt.Errorf("Restoring invalid flags: %s", e)
			}
			return nil
		}
	}
	if !n.setConditions {
		e = p.SetCPSR(previousConditions)
		if e != nil {
			return e
		}
	}
	return nil
}

func (n *PSRTransferInstruction) Emulate(p ARMProcessor) error {
	var value uint32
	var e error
	if !n.Condition().IsMet(p) {
		return nil
	}
	if !n.writePSR {
		if n.rd.Register() == 15 {
			return fmt.Errorf("Invalid mrs destination register")
		}
		if n.useCPSR {
			value, e = p.GetCPSR()
		} else {
			value, e = p.GetSPSR()
		}
		if e != nil {
			return e
		}
		p.SetRegister(n.rd, value)
		return nil
	}
	if n.isImmediate {
		r := n.rotate << 1
		value = uint32(n.immediate)
		value = (value >> r) | (value << (32 - r))
	} else {
		if n.rm.Register() == 15 {
			return fmt.Errorf("Invalid msr source register")
		}
		value, _ = p.GetRegister(n.rm)
	}
	if n.flagsOnly {
		var currentPSR uint32
		if n.useCPSR {
			currentPSR, e = p.GetCPSR()
		} else {
			currentPSR, e = p.GetSPSR()
		}
		if e != nil {
			return e
		}
		value = (value & 0xf0000000) | (currentPSR & 0x0fffffff)
	}
	if n.useCPSR {
		e = p.SetCPSR(value)
	} else {
		e = p.SetSPSR(value)
	}
	return e
}

func (n *MultiplyInstruction) Emulate(p ARMProcessor) error {
	if !n.Condition().IsMet(p) {
		return nil
	}
	a, _ := p.GetRegister(n.rm)
	b, _ := p.GetRegister(n.rs)
	if !n.isLongMultiply {
		result := uint32(a * b)
		if n.accumulate {
			c, _ := p.GetRegister(n.rn)
			result += c
		}
		p.SetRegister(n.rd, result)
		if n.setConditions {
			p.SetNegative((result & 0x80000000) != 0)
			p.SetZero(result == 0)
		}
		return nil
	}
	toAdd := uint64(0)
	if n.accumulate {
		highBits, _ := p.GetRegister(n.rdHigh)
		lowBits, _ := p.GetRegister(n.rdLow)
		toAdd = (uint64(highBits) << 32) | uint64(lowBits)
	}
	result := uint64(0)
	if n.signed {
		signedResult := int64(int32(a)) * int64(int32(b))
		signedResult += int64(toAdd)
		result = uint64(signedResult)
	} else {
		result = uint64(a) * uint64(b)
		result += toAdd
	}
	p.SetRegister(n.rdLow, uint32(result&0xffffffff))
	p.SetRegister(n.rdHigh, uint32(result>>32))
	if n.setConditions {
		p.SetNegative((result >> 63) != 0)
		p.SetZero(result == 0)
	}
	return nil
}

func (n *SingleDataSwapInstruction) Emulate(p ARMProcessor) error {
	var e error
	if !n.condition.IsMet(p) {
		return nil
	}
	address, _ := p.GetRegister(n.rn)
	// TODO: Implement a memory locking mechanism to use here?
	memory := p.GetMemoryInterface()
	if n.byteQuantity {
		value, e := memory.ReadMemoryByte(address)
		if e != nil {
			return e
		}
		toWrite, _ := p.GetRegister(n.rm)
		p.SetRegister(n.rd, uint32(value))
		e = memory.WriteMemoryByte(address, uint8(toWrite))
		return e
	}
	value, e := memory.ReadMemoryWord(address)
	if e != nil {
		return e
	}
	toWrite, _ := p.GetRegister(n.rm)
	p.SetRegister(n.rd, value)
	e = memory.WriteMemoryWord(address, toWrite)
	return e
}

func (n *BranchExchangeInstruction) Emulate(p ARMProcessor) error {
	var e error
	if !n.condition.IsMet(p) {
		return nil
	}
	destination, _ := p.GetRegister(n.rn)
	if (destination & 1) == 1 {
		e = p.SetTHUMBMode(true)
		if e != nil {
			return e
		}
	}
	p.SetRegisterNumber(15, destination)
	return nil
}

func (n *HalfwordDataTransferInstruction) Emulate(p ARMProcessor) error {
	var e error
	if !n.condition.IsMet(p) {
		return nil
	}
	memory := p.GetMemoryInterface()
	var offset uint32
	if n.isImmediate {
		offset = uint32(n.offset)
	} else {
		offset, _ = p.GetRegister(n.rm)
	}
	base, _ := p.GetRegister(n.rn)
	if n.rn.Register() == 15 {
		base += 4
	}
	if n.preindex {
		if n.up {
			base += offset
		} else {
			base -= offset
		}
	}
	var data uint32
	if n.load {
		if n.halfword {
			h, e := memory.ReadMemoryHalfword(base)
			if e != nil {
				return e
			}
			if n.signed {
				data = uint32(int32(int16(h)))
			} else {
				data = uint32(h)
			}
		} else {
			b, e := memory.ReadMemoryByte(base)
			if e != nil {
				return e
			}
			if n.signed {
				data = uint32(int32(int8(b)))
			} else {
				data = uint32(b)
			}
		}
		p.SetRegister(n.rd, data)
	} else {
		data, _ = p.GetRegister(n.rd)
		if n.rd.Register() == 15 {
			data += 8
		}
		e = memory.WriteMemoryHalfword(base, uint16(data&0xffff))
		if e != nil {
			return e
		}
	}
	if !n.preindex {
		if n.up {
			p.SetRegister(n.rn, base+offset)
		} else {
			p.SetRegister(n.rn, base-offset)
		}
	} else if n.writeBack {
		p.SetRegister(n.rn, base)
	}
	return nil
}

func (n *SingleDataTransferInstruction) Emulate(p ARMProcessor) error {
	var e error
	if !n.Condition().IsMet(p) {
		return nil
	}
	memory := p.GetMemoryInterface()
	var offset uint32
	if n.immediateOffset {
		offset = uint32(n.offset)
	} else {
		if n.shift.UseRegister() {
			return fmt.Errorf("Register-specified shift not allowed.")
		}
		if n.rm.Register() == 15 {
			return fmt.Errorf("Can't use r15 as offset in data transfer.")
		}
		offsetRegister, _ := p.GetRegister(n.rm)
		offset, e = n.shift.Apply(offsetRegister, p)
		if e != nil {
			return e
		}
	}
	base, _ := p.GetRegister(n.rn)
	if n.rn.Register() == 15 {
		base += 4
	}
	if n.preindex {
		if n.up {
			base += offset
		} else {
			base -= offset
		}
	}
	if n.load {
		var loadedValue uint32
		if n.byteQuantity {
			var byteValue uint8
			byteValue, e = memory.ReadMemoryByte(base)
			loadedValue = uint32(byteValue)
		} else {
			loadedValue, e = memory.ReadMemoryWord(base)
		}
		if e != nil {
			return e
		}
		p.SetRegister(n.rd, loadedValue)
	} else {
		toStore, e := p.GetRegister(n.rd)
		if n.rd.Register() == 15 {
			toStore += 8
		}
		if n.byteQuantity {
			e = memory.WriteMemoryByte(base, uint8(toStore))
		} else {
			e = memory.WriteMemoryWord(base, toStore)
		}
		if e != nil {
			return e
		}
	}
	if !n.preindex {
		if n.rn.Register() == 15 {
			return fmt.Errorf("r15 is incompatible with postindexing")
		}
		if n.up {
			p.SetRegister(n.rn, base+offset)
		} else {
			p.SetRegister(n.rn, base-offset)
		}
		// We probably don't need to do anything with the 'W' bit here, for
		// now (ldrt instruction, etc.)
	} else if n.writeBack {
		if n.rn.Register() == 15 {
			return fmt.Errorf("r15 is incompatible with writeback")
		}
		p.SetRegister(n.rn, base)
	}
	return nil
}

func (n *BlockDataTransferInstruction) blockDataStore(p ARMProcessor) error {
	var e error
	bits := n.registerList
	toStore := make([]uint32, 0, 16)
	for i := 0; (i < 16) && (bits != 0); i++ {
		if (bits & 1) == 1 {
			var registerContents uint32
			if n.forceUser {
				registerContents, e = p.GetUserRegisterNumber(uint8(i))
			} else {
				registerContents, e = p.GetRegisterNumber(uint8(i))
			}
			if e != nil {
				return e
			}
			toStore = append(toStore, registerContents)
		}
		bits = bits >> 1
	}
	// When storing "down", the values will be stored in the opposite order
	if !n.up {
		for i, j := 0, len(toStore)-1; i < j; i, j = i+1, j-1 {
			toStore[i], toStore[j] = toStore[j], toStore[i]
		}
	}
	baseAddress, _ := p.GetRegister(n.rn)
	memory := p.GetMemoryInterface()
	for _, value := range toStore {
		if n.preindex {
			if n.up {
				baseAddress += 4
			} else {
				baseAddress -= 4
			}
		}
		e = memory.WriteMemoryWord(baseAddress, value)
		if e != nil {
			return e
		}
		if !n.preindex {
			if n.up {
				baseAddress += 4
			} else {
				baseAddress -= 4
			}
		}
	}
	if n.writeBack {
		p.SetRegister(n.rn, baseAddress)
	}
	return nil
}

func (n *BlockDataTransferInstruction) blockDataLoad(p ARMProcessor) error {
	loadedBase := false
	useUserBank := n.forceUser && ((n.registerList & 0x8000) == 0)
	bits := n.registerList
	toRead := make([]uint8, 0, 16)
	for i := 0; (i < 16) && (bits != 0); i++ {
		if (bits & 1) == 1 {
			if n.rn.Register() == uint8(i) {
				loadedBase = true
			}
			toRead = append(toRead, uint8(i))
		}
		bits = bits >> 1
	}
	if !n.up {
		for i, j := 0, len(toRead)-1; i < j; i, j = i+1, j-1 {
			toRead[i], toRead[j] = toRead[j], toRead[i]
		}
	}
	baseAddress, _ := p.GetRegister(n.rn)
	memory := p.GetMemoryInterface()
	for _, registerNumber := range toRead {
		if n.preindex {
			if n.up {
				baseAddress += 4
			} else {
				baseAddress -= 4
			}
		}
		value, e := memory.ReadMemoryWord(baseAddress)
		if e != nil {
			return e
		}
		if useUserBank {
			e = p.SetUserRegisterNumber(registerNumber, value)
		} else {
			e = p.SetRegisterNumber(registerNumber, value)
		}
		if e != nil {
			return e
		}
		if !n.preindex {
			if n.up {
				baseAddress += 4
			} else {
				baseAddress -= 4
			}
		}
	}
	if n.writeBack && !loadedBase {
		p.SetRegister(n.rn, baseAddress)
	}
	return nil
}

func (n *BlockDataTransferInstruction) Emulate(p ARMProcessor) error {
	var e error
	if !n.condition.IsMet(p) {
		return nil
	}
	if n.load {
		e = n.blockDataLoad(p)
	} else {
		e = n.blockDataStore(p)
	}
	if e != nil {
		return e
	}
	if n.forceUser && n.load && ((n.registerList & 0x8000) != 0) {
		savedStatus, e := p.GetSPSR()
		if e != nil {
			return fmt.Errorf("Can't get SPSR in block data transfer: %s", e)
		}
		e = p.SetCPSR(savedStatus)
		if e != nil {
			return fmt.Errorf("Can't set CPSR in block data transfer: %s", e)
		}
	}
	return nil
}

func (n *BranchInstruction) Emulate(p ARMProcessor) error {
	if !n.condition.IsMet(p) {
		return nil
	}
	value, _ := p.GetRegisterNumber(15)
	pc := int32(value)
	if n.link {
		p.SetRegisterNumber(14, uint32(pc))
	}
	// Sign-extend the offset and shift it left 2 bits.
	offset := n.offset << 8
	offset = offset >> 6
	pc += 4 + offset
	p.SetRegisterNumber(15, uint32(pc))
	return nil
}

func (n *CoprocDataTransferInstruction) Emulate(p ARMProcessor) error {
	var e error
	if !n.condition.IsMet(p) {
		return nil
	}
	address, _ := p.GetRegister(n.rn)
	offset := uint32(n.offset) << 2
	if n.preindex {
		if n.up {
			address += offset
		} else {
			address -= offset
		}
	}
	for _, c := range p.GetCoprocessors() {
		if c.Number() != n.coprocNumber {
			continue
		}
		e = c.DataTransfer(p, n.raw, address)
		if e != nil {
			return fmt.Errorf("Coprocessor data transfer error: %s", e)
		}
		break
	}
	if n.writeBack {
		if !n.preindex {
			if n.up {
				address += offset
			} else {
				address -= offset
			}
		}
		p.SetRegister(n.rn, address)
	}
	return nil
}

func (n *CoprocDataOperationInstruction) Emulate(p ARMProcessor) error {
	var e error
	if !n.condition.IsMet(p) {
		return nil
	}
	for _, c := range p.GetCoprocessors() {
		if c.Number() != n.coprocNumber {
			continue
		}
		e = c.Operation(p, n.raw)
		if e != nil {
			return fmt.Errorf("Coprocessor operation error: %s", e)
		}
		break
	}
	return nil
}

func (n *CoprocRegisterTransferInstruction) Emulate(p ARMProcessor) error {
	var e error
	if !n.condition.IsMet(p) {
		return nil
	}
	for _, c := range p.GetCoprocessors() {
		if c.Number() != n.coprocNumber {
			continue
		}
		e = c.RegisterTransfer(p, n.raw, n.rd, n.load)
		if e != nil {
			return fmt.Errorf("Coprocessor register transfer error: %s", e)
		}
		break
	}
	return nil
}

func (n *SoftwareInterruptInstruction) Emulate(p ARMProcessor) error {
	var e error
	if !n.condition.IsMet(p) {
		return nil
	}
	currentPC, _ := p.GetRegisterNumber(15)
	e = p.SetMode(0x13)
	if e != nil {
		return e
	}
	p.SetRegisterNumber(14, currentPC)
	p.SetRegisterNumber(15, 0x8)
	return nil
}
