package arm_emulate

import (
	"fmt"
)

func (n *DataProcessingInstruction) evaluateSecondOperand(
	p ARMProcessor) (uint32, error) {
	if n.IsImmediate {
		r := n.Rotate << 1
		value := uint32(n.Immediate)
		return (value >> r) | (value << (32 - r)), nil
	}
	value, _ := p.GetRegister(n.Rm)
	if n.Rm == 15 {
		value += 4
		if n.Shift.UseRegister() {
			value += 4
		}
	}
	return n.Shift.Apply(value, p)
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
	operand1, _ := p.GetRegister(n.Rn)
	if n.Rn == 15 {
		operand1 += 4
		if !n.IsImmediate && n.Shift.UseRegister() {
			operand1 += 4
		}
	}
	result, writeResult, e := n.Opcode.Evaluate(operand1, operand2, p)
	if e != nil {
		return e
	}
	if writeResult {
		p.SetRegister(n.Rd, result)
		// Writes to r15 with the S bit set is an atomic operation to switch
		// modes (not valid in user mode)
		if (n.Rd == 15) && n.SetConditions {
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
	if !n.SetConditions {
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
	if !n.WritePSR {
		if n.Rd == 15 {
			return fmt.Errorf("Invalid mrs destination register")
		}
		if n.UseCPSR {
			value, e = p.GetCPSR()
		} else {
			value, e = p.GetSPSR()
		}
		if e != nil {
			return e
		}
		p.SetRegister(n.Rd, value)
		return nil
	}
	if n.IsImmediate {
		r := n.Rotate << 1
		value = uint32(n.Immediate)
		value = (value >> r) | (value << (32 - r))
	} else {
		if n.Rm == 15 {
			return fmt.Errorf("Invalid msr source register")
		}
		value, _ = p.GetRegister(n.Rm)
	}
	// Prevent user-mode from changing anything except flags.
	if n.FlagsOnly || (p.GetMode() == userMode) {
		var currentPSR uint32
		if n.UseCPSR {
			currentPSR, e = p.GetCPSR()
		} else {
			currentPSR, e = p.GetSPSR()
		}
		if e != nil {
			return e
		}
		value = (value & 0xf0000000) | (currentPSR & 0x0fffffff)
	}
	if n.UseCPSR {
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
	a, _ := p.GetRegister(n.Rm)
	b, _ := p.GetRegister(n.Rs)
	if !n.IsLongMultiply {
		result := uint32(a * b)
		if n.Accumulate {
			c, _ := p.GetRegister(n.Rn)
			result += c
		}
		p.SetRegister(n.Rd, result)
		if n.SetConditions {
			p.SetNegative((result & 0x80000000) != 0)
			p.SetZero(result == 0)
		}
		return nil
	}
	toAdd := uint64(0)
	if n.Accumulate {
		highBits, _ := p.GetRegister(n.RdHigh)
		lowBits, _ := p.GetRegister(n.RdLow)
		toAdd = (uint64(highBits) << 32) | uint64(lowBits)
	}
	result := uint64(0)
	if n.Signed {
		signedResult := int64(int32(a)) * int64(int32(b))
		signedResult += int64(toAdd)
		result = uint64(signedResult)
	} else {
		result = uint64(a) * uint64(b)
		result += toAdd
	}
	p.SetRegister(n.RdLow, uint32(result&0xffffffff))
	p.SetRegister(n.RdHigh, uint32(result>>32))
	if n.SetConditions {
		p.SetNegative((result >> 63) != 0)
		p.SetZero(result == 0)
	}
	return nil
}

func (n *SingleDataSwapInstruction) Emulate(p ARMProcessor) error {
	var e error
	if !n.Condition().IsMet(p) {
		return nil
	}
	address, _ := p.GetRegister(n.Rn)
	// TODO: Implement a memory locking mechanism to use here?
	memory := p.GetMemoryInterface()
	if n.ByteQuantity {
		value, e := memory.ReadMemoryByte(address)
		if e != nil {
			return e
		}
		toWrite, _ := p.GetRegister(n.Rm)
		p.SetRegister(n.Rd, uint32(value))
		e = memory.WriteMemoryByte(address, uint8(toWrite))
		return e
	}
	value, e := memory.ReadMemoryWord(address)
	if e != nil {
		return e
	}
	toWrite, _ := p.GetRegister(n.Rm)
	p.SetRegister(n.Rd, value)
	e = memory.WriteMemoryWord(address, toWrite)
	return e
}

func (n *BranchExchangeInstruction) Emulate(p ARMProcessor) error {
	var e error
	if !n.Condition().IsMet(p) {
		return nil
	}
	destination, _ := p.GetRegister(n.Rn)
	if (destination & 1) == 1 {
		e = p.SetTHUMBMode(true)
		if e != nil {
			return e
		}
	}
	p.SetRegister(15, destination)
	return nil
}

func (n *HalfwordDataTransferInstruction) Emulate(p ARMProcessor) error {
	var e error
	if !n.Condition().IsMet(p) {
		return nil
	}
	memory := p.GetMemoryInterface()
	var offset uint32
	if n.IsImmediate {
		offset = uint32(n.Offset)
	} else {
		offset, _ = p.GetRegister(n.Rm)
	}
	base, _ := p.GetRegister(n.Rn)
	if n.Rn == 15 {
		base += 4
	}
	if n.Preindex {
		if n.Up {
			base += offset
		} else {
			base -= offset
		}
	}
	var data uint32
	if n.Load {
		if n.Halfword {
			h, e := memory.ReadMemoryHalfword(base)
			if e != nil {
				return e
			}
			if n.Signed {
				data = uint32(int32(int16(h)))
			} else {
				data = uint32(h)
			}
		} else {
			b, e := memory.ReadMemoryByte(base)
			if e != nil {
				return e
			}
			if n.Signed {
				data = uint32(int32(int8(b)))
			} else {
				data = uint32(b)
			}
		}
		p.SetRegister(n.Rd, data)
	} else {
		data, _ = p.GetRegister(n.Rd)
		if n.Rd == 15 {
			data += 8
		}
		e = memory.WriteMemoryHalfword(base, uint16(data&0xffff))
		if e != nil {
			return e
		}
	}
	if !n.Preindex {
		if n.Up {
			p.SetRegister(n.Rn, base+offset)
		} else {
			p.SetRegister(n.Rn, base-offset)
		}
	} else if n.WriteBack {
		p.SetRegister(n.Rn, base)
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
	if n.ImmediateOffset {
		offset = uint32(n.Offset)
	} else {
		if n.Shift.UseRegister() {
			return fmt.Errorf("Register-specified shift not allowed.")
		}
		if n.Rm == 15 {
			return fmt.Errorf("Can't use r15 as offset in data transfer.")
		}
		offsetRegister, _ := p.GetRegister(n.Rm)
		offset, e = n.Shift.Apply(offsetRegister, p)
		if e != nil {
			return e
		}
	}
	base, _ := p.GetRegister(n.Rn)
	if n.Rn == 15 {
		base += 4
	}
	if n.Preindex {
		if n.Up {
			base += offset
		} else {
			base -= offset
		}
	}
	if n.Load {
		var loadedValue uint32
		if n.ByteQuantity {
			var byteValue uint8
			byteValue, e = memory.ReadMemoryByte(base)
			loadedValue = uint32(byteValue)
		} else {
			loadedValue, e = memory.ReadMemoryWord(base)
		}
		if e != nil {
			return e
		}
		p.SetRegister(n.Rd, loadedValue)
	} else {
		toStore, e := p.GetRegister(n.Rd)
		if n.Rd == 15 {
			toStore += 8
		}
		if n.ByteQuantity {
			e = memory.WriteMemoryByte(base, uint8(toStore))
		} else {
			e = memory.WriteMemoryWord(base, toStore)
		}
		if e != nil {
			return e
		}
	}
	if !n.Preindex {
		if n.Rn == 15 {
			return fmt.Errorf("r15 is incompatible with postindexing")
		}
		if n.Up {
			p.SetRegister(n.Rn, base+offset)
		} else {
			p.SetRegister(n.Rn, base-offset)
		}
		// We probably don't need to do anything with the 'W' bit here, for
		// now (ldrt instruction, etc.)
	} else if n.WriteBack {
		if n.Rn == 15 {
			return fmt.Errorf("r15 is incompatible with writeback")
		}
		p.SetRegister(n.Rn, base)
	}
	return nil
}

func (n *BlockDataTransferInstruction) blockDataStore(p ARMProcessor) error {
	var e error
	bits := n.RegisterList
	toStore := make([]uint32, 0, 16)
	for i := 0; (i < 16) && (bits != 0); i++ {
		if (bits & 1) == 1 {
			var registerContents uint32
			if n.ForceUser {
				registerContents, e = p.GetUserRegister(ARMRegister(i))
			} else {
				registerContents, e = p.GetRegister(ARMRegister(i))
			}
			if e != nil {
				return e
			}
			toStore = append(toStore, registerContents)
		}
		bits = bits >> 1
	}
	// When storing "down", the values will be stored in the opposite order
	if !n.Up {
		for i, j := 0, len(toStore)-1; i < j; i, j = i+1, j-1 {
			toStore[i], toStore[j] = toStore[j], toStore[i]
		}
	}
	baseAddress, _ := p.GetRegister(n.Rn)
	memory := p.GetMemoryInterface()
	for _, value := range toStore {
		if n.Preindex {
			if n.Up {
				baseAddress += 4
			} else {
				baseAddress -= 4
			}
		}
		e = memory.WriteMemoryWord(baseAddress, value)
		if e != nil {
			return e
		}
		if !n.Preindex {
			if n.Up {
				baseAddress += 4
			} else {
				baseAddress -= 4
			}
		}
	}
	if n.WriteBack {
		p.SetRegister(n.Rn, baseAddress)
	}
	return nil
}

func (n *BlockDataTransferInstruction) blockDataLoad(p ARMProcessor) error {
	loadedBase := false
	useUserBank := n.ForceUser && ((n.RegisterList & 0x8000) == 0)
	bits := n.RegisterList
	toRead := make([]uint8, 0, 16)
	for i := 0; (i < 16) && (bits != 0); i++ {
		if (bits & 1) == 1 {
			if n.Rn == ARMRegister(i) {
				loadedBase = true
			}
			toRead = append(toRead, uint8(i))
		}
		bits = bits >> 1
	}
	if !n.Up {
		for i, j := 0, len(toRead)-1; i < j; i, j = i+1, j-1 {
			toRead[i], toRead[j] = toRead[j], toRead[i]
		}
	}
	baseAddress, _ := p.GetRegister(n.Rn)
	memory := p.GetMemoryInterface()
	for _, registerNumber := range toRead {
		if n.Preindex {
			if n.Up {
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
			e = p.SetUserRegister(ARMRegister(registerNumber), value)
		} else {
			e = p.SetRegister(ARMRegister(registerNumber), value)
		}
		if e != nil {
			return e
		}
		if !n.Preindex {
			if n.Up {
				baseAddress += 4
			} else {
				baseAddress -= 4
			}
		}
	}
	if n.WriteBack && !loadedBase {
		p.SetRegister(n.Rn, baseAddress)
	}
	return nil
}

func (n *BlockDataTransferInstruction) Emulate(p ARMProcessor) error {
	var e error
	if !n.Condition().IsMet(p) {
		return nil
	}
	if n.Load {
		e = n.blockDataLoad(p)
	} else {
		e = n.blockDataStore(p)
	}
	if e != nil {
		return e
	}
	if n.ForceUser && n.Load && ((n.RegisterList & 0x8000) != 0) {
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
	if !n.Condition().IsMet(p) {
		return nil
	}
	value, _ := p.GetRegister(15)
	pc := int32(value)
	if n.Link {
		p.SetRegister(14, uint32(pc))
	}
	// Sign-extend the offset and shift it left 2 bits.
	offset := n.Offset << 8
	offset = offset >> 6
	pc += 4 + offset
	p.SetRegister(15, uint32(pc))
	return nil
}

func (n *CoprocDataTransferInstruction) Emulate(p ARMProcessor) error {
	var e error
	if !n.Condition().IsMet(p) {
		return nil
	}
	address, _ := p.GetRegister(n.Rn)
	offset := uint32(n.Offset) << 2
	if n.Preindex {
		if n.Up {
			address += offset
		} else {
			address -= offset
		}
	}
	for _, c := range p.GetCoprocessors() {
		if c.Number() != n.CoprocNumber {
			continue
		}
		e = c.DataTransfer(p, n.raw, address)
		if e != nil {
			return fmt.Errorf("Coprocessor data transfer error: %s", e)
		}
		break
	}
	if n.WriteBack {
		if !n.Preindex {
			if n.Up {
				address += offset
			} else {
				address -= offset
			}
		}
		p.SetRegister(n.Rn, address)
	}
	return nil
}

func (n *CoprocDataOperationInstruction) Emulate(p ARMProcessor) error {
	var e error
	if !n.Condition().IsMet(p) {
		return nil
	}
	for _, c := range p.GetCoprocessors() {
		if c.Number() != n.CoprocNumber {
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
	if !n.Condition().IsMet(p) {
		return nil
	}
	for _, c := range p.GetCoprocessors() {
		if c.Number() != n.CoprocNumber {
			continue
		}
		e = c.RegisterTransfer(p, n.raw, n.Rd, n.Load)
		if e != nil {
			return fmt.Errorf("Coprocessor register transfer error: %s", e)
		}
		break
	}
	return nil
}

func (n *SoftwareInterruptInstruction) Emulate(p ARMProcessor) error {
	var e error
	if !n.Condition().IsMet(p) {
		return nil
	}
	currentPC, _ := p.GetRegister(15)
	e = p.SetMode(0x13)
	if e != nil {
		return e
	}
	p.SetRegister(14, currentPC)
	p.SetRegister(15, 0x8)
	return nil
}
