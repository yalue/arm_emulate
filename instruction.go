package arm_emulate

import (
	"fmt"
)

const (
	singleDataSwapMask       uint32 = 0x0fb00ff0
	singleDataSwapSet        uint32 = 0x01000090
	branchExchangeMask       uint32 = 0x0ffffff0
	branchExchangeSet        uint32 = 0x012fff10
	singleDataTransferMask   uint32 = 0x04000000
	multiplyMask             uint32 = 0x0f0000f0
	multiplySet              uint32 = 0x00000090
	halfwordDataTransferMask uint32 = 0x0e000090
	halfwordDataTransferSet  uint32 = 0x00000090
	dataProcessingMask       uint32 = 0x0c000000
	dataProcessingSet        uint32 = 0x00000000
	psrTransferMask          uint32 = 0x0d980000
	psrTransferSet           uint32 = 0x01080000
	undefinedMask            uint32 = 0x0e000010
	undefinedSet             uint32 = 0x06000010
)

type ARMInstruction interface {
	fmt.Stringer
	Raw() uint32
	Condition() ARMCondition
	Emulate(p ARMProcessor) error
}

type basicARMInstruction struct {
	raw       uint32
	condition ARMCondition
}

func (n *basicARMInstruction) Raw() uint32 {
	return n.raw
}

func (n *basicARMInstruction) Condition() ARMCondition {
	return n.condition
}

func (n *basicARMInstruction) Emulate(p ARMProcessor) error {
	return fmt.Errorf("Emulation not implemented for 0x%08x", n.raw)
}

func (n *basicARMInstruction) String() string {
	return fmt.Sprintf("data: 0x%08x", n.raw)
}

type DataProcessingInstruction struct {
	basicARMInstruction
	opcode        ARMDataProcessingOpcode
	rm            ARMRegister
	shift         ARMShift
	immediate     uint8
	rotate        uint8
	rd            ARMRegister
	rn            ARMRegister
	setConditions bool
	isImmediate   bool
}

func (n *DataProcessingInstruction) secondOperand() string {
	if n.isImmediate {
		r := n.rotate << 1
		value := uint32(n.immediate)
		value = (value >> r) | (value << (32 - r))
		return fmt.Sprintf("%d", value)
	}
	toReturn := n.rm.String()
	if n.shift.UseRegister() || (n.shift.Amount() != 0) {
		toReturn += " "
	}
	return fmt.Sprintf("%s%s", toReturn, n.shift)
}

func (n *DataProcessingInstruction) String() string {
	prefix := n.opcode.String()
	prefix += n.condition.String()
	opcodeValue := n.opcode.Value()
	switch opcodeValue {
	case movARMOpcode, mvnARMOpcode:
		if n.setConditions {
			prefix += "s"
		}
		return fmt.Sprintf("%s %s, %s", prefix, n.rd, n.secondOperand())
	case tstARMOpcode, teqARMOpcode, cmnARMOpcode, cmpARMOpcode:
		return fmt.Sprintf("%s %s, %s", prefix, n.rn, n.secondOperand())
	}
	if n.setConditions {
		prefix += "s"
	}
	return fmt.Sprintf("%s %s, %s, %s", prefix, n.rd, n.rn, n.secondOperand())
}

type PSRTransferInstruction struct {
	basicARMInstruction
	rm          ARMRegister
	rd          ARMRegister
	isImmediate bool
	writePSR    bool
	useCPSR     bool
	flagsOnly   bool
	immediate   uint8
	rotate      uint8
}

func (n *PSRTransferInstruction) String() string {
	var usedPSR string
	if n.useCPSR {
		usedPSR = "cpsr"
	} else {
		usedPSR = "spsr"
	}
	if !n.writePSR {
		return fmt.Sprintf("mrs%s %s, %s", n.condition, n.rd, usedPSR)
	}
	if !n.flagsOnly {
		return fmt.Sprintf("msr%s %s, %s", n.condition, n.rm, usedPSR)
	}
	usedPSR += "_flags"
	if n.isImmediate {
		r := n.rotate << 1
		value := uint32(n.immediate)
		value = (value >> r) | (value << (32 - r))
		return fmt.Sprintf("msr%s %s, %d", n.condition, usedPSR, value)
	}
	return fmt.Sprintf("msr%s %s, %s", n.condition, usedPSR, n.rm)
}

type MultiplyInstruction struct {
	basicARMInstruction
	isLongMultiply bool
	rm             ARMRegister
	rn             ARMRegister
	rs             ARMRegister
	rd             ARMRegister
	rdLow          ARMRegister
	rdHigh         ARMRegister
	setConditions  bool
	accumulate     bool
	signed         bool
}

func (n *MultiplyInstruction) String() string {
	var start string
	if n.accumulate {
		start = "mla"
	} else {
		start = "mul"
	}
	if n.isLongMultiply {
		if n.signed {
			start = "s" + start + "l"
		} else {
			start = "u" + start + "l"
		}
	}
	start += n.condition.String()
	if n.setConditions {
		start += "s"
	}
	if n.isLongMultiply {
		return fmt.Sprintf("%s %s, %s, %s, %s", start, n.rdLow, n.rdHigh, n.rm,
			n.rs)
	}
	if !n.accumulate {
		return fmt.Sprintf("%s %s, %s, %s", start, n.rd, n.rm, n.rs)
	}
	return fmt.Sprintf("%s %s, %s, %s, %s", start, n.rd, n.rm, n.rs, n.rn)
}

type SingleDataSwapInstruction struct {
	basicARMInstruction
	rm           ARMRegister
	rn           ARMRegister
	rd           ARMRegister
	byteQuantity bool
}

func (n *SingleDataSwapInstruction) String() string {
	start := "swp"
	start += n.condition.String()
	if n.byteQuantity {
		start += "b"
	}
	return fmt.Sprintf("%s %s, %s, [%s]", start, n.rd, n.rm, n.rn)
}

type BranchExchangeInstruction struct {
	basicARMInstruction
	rn ARMRegister
}

func (n *BranchExchangeInstruction) String() string {
	return fmt.Sprintf("bx%s %s", n.condition, n.rn)
}

type HalfwordDataTransferInstruction struct {
	basicARMInstruction
	isImmediate bool
	halfword    bool
	signed      bool
	rm          ARMRegister
	rn          ARMRegister
	rd          ARMRegister
	offset      uint8
	load        bool
	writeBack   bool
	up          bool
	preindex    bool
}

func (n *HalfwordDataTransferInstruction) String() string {
	var start string
	if n.load {
		start = "ldr"
	} else {
		start = "str"
	}
	start += n.condition.String()
	if n.signed {
		start += "s"
	}
	if n.halfword {
		start += "h"
	} else {
		start += "b"
	}
	start += " " + n.rd.String() + ","
	offset := int(n.offset)
	offsetReg := n.rm.String()
	if n.rn.Register() == 15 {
		offset += 8
	}
	if !n.up {
		offset = -offset
		offsetReg = "-" + offsetReg
	}
	if n.isImmediate && n.preindex && !n.writeBack && (n.rn.Register() == 15) {
		return fmt.Sprintf("%s %d", start, offset)
	}
	if n.preindex {
		postfix := ""
		if n.writeBack {
			postfix = "!"
		}
		if n.isImmediate {
			if n.offset == 0 {
				return fmt.Sprintf("%s [%s]%s", start, n.rn, postfix)
			}
			return fmt.Sprintf("%s [%s, %d]%s", start, n.rn, offset, postfix)
		}
		return fmt.Sprintf("%s [%s, %s]%s", start, n.rn, offsetReg, postfix)
	}
	if n.isImmediate {
		return fmt.Sprintf("%s [%s], %d", start, n.rn, offset)
	}
	return fmt.Sprintf("%s [%s], %s", start, n.rn, offsetReg)
}

type SingleDataTransferInstruction struct {
	basicARMInstruction
	rn              ARMRegister
	rd              ARMRegister
	rm              ARMRegister
	shift           ARMShift
	offset          uint16
	load            bool
	writeBack       bool
	byteQuantity    bool
	up              bool
	preindex        bool
	immediateOffset bool
}

func (n *SingleDataTransferInstruction) String() string {
	var start string
	if n.load {
		start = "ldr"
	} else {
		start = "str"
	}
	start += n.condition.String()
	if n.byteQuantity {
		start += "b"
	}
	if !n.preindex && n.writeBack {
		start += "t"
	}
	start += " " + n.rd.String() + ","
	upString := ""
	if !n.up {
		upString = "-"
	}
	shiftString := ""
	if !n.immediateOffset && (n.shift.Amount() != 0) {
		shiftString = ", " + n.shift.String()
	}
	offset := int(n.offset)
	if n.rn.Register() == 15 {
		offset += 8
	}
	if n.preindex {
		postfix := ""
		if n.writeBack {
			postfix = "!"
		}
		if n.immediateOffset {
			if (n.rn.Register() == 15) && (offset != 0) {
				return fmt.Sprintf("%s %s%d", start, upString, offset)
			}
			if offset == 0 {
				return fmt.Sprintf("%s [%s]%s", start, n.rn, postfix)
			}
			return fmt.Sprintf("%s [%s, %s%d]%s", start, n.rn, upString,
				offset, postfix)
		}
		return fmt.Sprintf("%s [%s, %s%s%s]%s", start, n.rn, upString, n.rm,
			shiftString, postfix)
	}
	if n.immediateOffset {
		return fmt.Sprintf("%s [%s], %s%d", start, n.rn, upString, offset)
	}
	return fmt.Sprintf("%s [%s], %s%s%s", start, n.rn, upString, n.rm,
		shiftString)
}

type UndefinedInstruction struct {
	basicARMInstruction
}

type BlockDataTransferInstruction struct {
	basicARMInstruction
	registerList uint16
	rn           ARMRegister
	load         bool
	writeBack    bool
	forceUser    bool
	up           bool
	preindex     bool
}

func (n *BlockDataTransferInstruction) listString() string {
	var s string
	consecutive := uint8(0)
	s = "{"
	registers := n.registerList
	// The 17th iteration will always be a 0 bit, but it is still necessary if
	// r15 was set.
	for i := uint8(0); i < 17; i++ {
		if (registers & 1) == 1 {
			consecutive++
		} else if consecutive != 0 {
			startRegister := i - consecutive
			endRegister := i - 1
			consecutive = 0
			if s[len(s)-1] != '{' {
				s += ", "
			}
			// If multiple consecutive registers, use "rx-ry" format, but use
			// "sp", "lr" or "pc" for single registers
			if startRegister == endRegister {
				s += NewARMRegister(endRegister).String()
			} else {
				s += fmt.Sprintf("r%d-r%d", startRegister, endRegister)
			}
		}
		registers = registers >> 1
	}
	s += "}"
	return s
}

func (n *BlockDataTransferInstruction) String() string {
	var start string
	if n.load {
		start = "ldm"
	} else {
		start = "stm"
	}
	// The mnemonic postfix depends on the u and p bits and stack usage
	if n.rn.Register() == 13 {
		if n.up {
			if n.preindex {
				start += "ed"
			} else {
				start += "fd"
			}
		} else {
			if n.preindex {
				start += "ea"
			} else {
				start += "fa"
			}
		}
	} else {
		if n.up {
			if n.preindex {
				start += "ib"
			} else {
				start += "ia"
			}
		} else {
			if n.preindex {
				start += "db"
			} else {
				start += "da"
			}
		}
	}
	start += " " + n.rn.String()
	if n.writeBack {
		start += "!"
	}
	start += ", " + n.listString()
	if n.forceUser {
		start += "^"
	}
	return start
}

type BranchInstruction struct {
	basicARMInstruction
	offset int32
	link   bool
}

func (n *BranchInstruction) String() string {
	start := "b"
	if n.link {
		start += "l"
	}
	start += n.condition.String()
	// Sign extend and shift right by 2 bits...
	offset := n.offset << 8
	offset = offset >> 6
	return fmt.Sprintf("%s %d", start, offset)
}

type CoprocDataTransferInstruction struct {
	basicARMInstruction
	rn           ARMRegister
	coprocNumber uint8
	coprocRd     uint8
	offset       uint8
	load         bool
	writeBack    bool
	longTransfer bool
	up           bool
	preindex     bool
}

func (n *CoprocDataTransferInstruction) String() string {
	var start string
	if n.load {
		start = "ldc"
	} else {
		start = "stc"
	}
	start += n.condition.String()
	if n.longTransfer {
		start += "l"
	}
	start += fmt.Sprintf(" p%d, c%d,", n.coprocNumber, n.coprocRd)
	offset := int(n.offset) << 2
	if !n.up {
		offset = -offset
	}
	if n.rn.Register() == 15 {
		offset += 8
	}
	if n.preindex {
		postfix := ""
		if n.writeBack {
			postfix = "!"
		}
		if !n.writeBack && (n.rn.Register() == 15) {
			return fmt.Sprintf("%s %d", start, offset)
		}
		if n.offset == 0 {
			return fmt.Sprintf("%s [%s]%s", start, n.rn, postfix)
		}
		return fmt.Sprintf("%s [%s, %d]%s", start, n.rn, offset, postfix)
	}
	if n.offset == 0 {
		return fmt.Sprintf("%s %s", start, n.rn)
	}
	return fmt.Sprintf("%s [%s], %d", start, n.rn, offset)
}

type CoprocDataOperationInstruction struct {
	basicARMInstruction
	coprocNumber uint8
	coprocOpcode uint8
	coprocInfo   uint8
	coprocRn     uint8
	coprocRd     uint8
	coprocRm     uint8
}

func (n *CoprocDataOperationInstruction) String() string {
	return fmt.Sprintf("cdp%s p%d, %d, c%d, c%d, c%d, %d", n.condition,
		n.coprocNumber, n.coprocOpcode, n.coprocRd, n.coprocRn, n.coprocRm,
		n.coprocInfo)
}

type CoprocRegisterTransferInstruction struct {
	basicARMInstruction
	rd            ARMRegister
	load          bool
	coprocNumber  uint8
	coprocOpcode  uint8
	coprocOperand uint8
	coprocRn      uint8
	coprocRm      uint8
}

func (n *CoprocRegisterTransferInstruction) String() string {
	var start string
	if n.load {
		start = "mrc"
	} else {
		start = "mcr"
	}
	start += n.condition.String()
	return fmt.Sprintf("%s p%d, %d, %s, c%d, c%d, %d", start, n.coprocNumber,
		n.coprocOpcode, n.rd, n.coprocRn, n.coprocRm, n.coprocOperand)
}

type SoftwareInterruptInstruction struct {
	basicARMInstruction
	comment uint32
}

func (n *SoftwareInterruptInstruction) String() string {
	return fmt.Sprintf("swi%s %08x", n.condition, n.comment)
}

func getCondition(raw uint32) ARMCondition {
	return NewARMCondition(uint8((raw >> 28) & 0xf))
}

func parseSoftwareInterruptInstruction(raw uint32) (ARMInstruction, error) {
	var toReturn SoftwareInterruptInstruction
	toReturn.raw = raw
	toReturn.condition = getCondition(raw)
	toReturn.comment = raw & 0x00ffffff
	return &toReturn, nil
}

func parseCoprocRegisterTransferInstruction(raw uint32) (ARMInstruction,
	error) {
	var toReturn CoprocRegisterTransferInstruction
	toReturn.raw = raw
	toReturn.condition = getCondition(raw)
	toReturn.rd = NewARMRegister(uint8((raw >> 12) & 0xf))
	toReturn.load = (raw & 0x100000) != 0
	toReturn.coprocNumber = uint8((raw >> 8) & 0xf)
	toReturn.coprocOpcode = uint8((raw >> 21) & 0x7)
	toReturn.coprocOperand = uint8((raw >> 5) & 0x7)
	toReturn.coprocRm = uint8(raw & 0xf)
	toReturn.coprocRn = uint8((raw >> 16) & 0xf)
	return &toReturn, nil
}

func parseCoprocDataOperationInstruction(raw uint32) (ARMInstruction, error) {
	var toReturn CoprocDataOperationInstruction
	toReturn.raw = raw
	toReturn.condition = getCondition(raw)
	toReturn.coprocNumber = uint8((raw >> 8) & 0xf)
	toReturn.coprocRm = uint8(raw & 0xf)
	toReturn.coprocRd = uint8((raw >> 12) & 0xf)
	toReturn.coprocRn = uint8((raw >> 16) & 0xf)
	toReturn.coprocOpcode = uint8((raw >> 20) & 0xf)
	toReturn.coprocInfo = uint8((raw >> 5) & 0x7)
	return &toReturn, nil
}

func parseCoprocDataTransferInstruction(raw uint32) (ARMInstruction, error) {
	var toReturn CoprocDataTransferInstruction
	toReturn.raw = raw
	toReturn.condition = getCondition(raw)
	toReturn.rn = NewARMRegister(uint8((raw >> 16) & 0xf))
	toReturn.offset = uint8(raw & 0xff)
	toReturn.coprocNumber = uint8((raw >> 8) & 0xf)
	toReturn.coprocRd = uint8((raw >> 12) & 0xf)
	toReturn.load = (raw & 0x100000) != 0
	toReturn.writeBack = (raw & 0x200000) != 0
	toReturn.longTransfer = (raw & 0x400000) != 0
	toReturn.up = (raw & 0x800000) != 0
	toReturn.preindex = (raw & 0x1000000) != 0
	return &toReturn, nil
}

func parseBranchInstruction(raw uint32) (ARMInstruction, error) {
	var toReturn BranchInstruction
	toReturn.raw = raw
	toReturn.condition = getCondition(raw)
	toReturn.offset = int32(raw) & int32(0x00ffffff)
	toReturn.link = (raw & 0x1000000) != 0
	return &toReturn, nil
}

func parseBlockDataTransferInstruction(raw uint32) (ARMInstruction, error) {
	var toReturn BlockDataTransferInstruction
	toReturn.raw = raw
	toReturn.condition = getCondition(raw)
	toReturn.registerList = uint16(raw & 0xffff)
	toReturn.rn = NewARMRegister(uint8((raw >> 16) & 0xf))
	toReturn.load = (raw & 0x100000) != 0
	toReturn.writeBack = (raw & 0x200000) != 0
	toReturn.forceUser = (raw & 0x400000) != 0
	toReturn.up = (raw & 0x800000) != 0
	toReturn.preindex = (raw & 0x1000000) != 0
	return &toReturn, nil
}

func parseUndefinedInstruction(raw uint32) (ARMInstruction, error) {
	var toReturn UndefinedInstruction
	toReturn.raw = raw
	toReturn.condition = getCondition(raw)
	return &toReturn, fmt.Errorf("Undefined instruction")
}

func parseSingleDataTransferInstruction(raw uint32) (ARMInstruction, error) {
	var toReturn SingleDataTransferInstruction
	toReturn.raw = raw
	toReturn.condition = getCondition(raw)
	toReturn.immediateOffset = (raw & 0x2000000) == 0
	if !toReturn.immediateOffset {
		toReturn.shift = NewARMShift(uint8((raw >> 4) & 0xff))
		// This shouldn't happen as along as the undefined instruction mask is
		// checked before the single data transfer instruction mask
		if toReturn.shift.UseRegister() {
			var errorInstruction UndefinedInstruction
			errorInstruction.raw = raw
			errorInstruction.condition = toReturn.condition
			return &errorInstruction, fmt.Errorf("Illegal shift")
		}
		toReturn.rm = NewARMRegister(uint8(raw & 0xf))
	} else {
		toReturn.offset = uint16(raw & 0xfff)
	}
	toReturn.rd = NewARMRegister(uint8((raw >> 12) & 0xf))
	toReturn.rn = NewARMRegister(uint8((raw >> 16) & 0xf))
	toReturn.load = (raw & 0x100000) != 0
	toReturn.writeBack = (raw & 0x200000) != 0
	toReturn.byteQuantity = (raw & 0x400000) != 0
	toReturn.up = (raw & 0x800000) != 0
	toReturn.preindex = (raw & 0x1000000) != 0
	return &toReturn, nil
}

func parseHalfwordDataTransferInstruction(raw uint32) (ARMInstruction, error) {
	var toReturn HalfwordDataTransferInstruction
	toReturn.raw = raw
	toReturn.condition = getCondition(raw)
	toReturn.isImmediate = (raw & 0x400000) != 0
	if toReturn.isImmediate {
		toReturn.offset = uint8((raw & 0xf) | ((raw >> 4) & 0xf0))
	} else {
		toReturn.rm = NewARMRegister(uint8(raw & 0xf))
	}
	toReturn.halfword = (raw & 0x20) != 0
	toReturn.signed = (raw & 0x40) != 0
	toReturn.rd = NewARMRegister(uint8((raw >> 12) & 0xf))
	toReturn.rn = NewARMRegister(uint8((raw >> 16) & 0xf))
	toReturn.load = (raw & 0x100000) != 0
	toReturn.writeBack = (raw & 0x200000) != 0
	toReturn.up = (raw & 0x800000) != 0
	toReturn.preindex = (raw & 0x1000000) != 0
	return &toReturn, nil
}

func parseBranchExchangeInstruction(raw uint32) (ARMInstruction, error) {
	var toReturn BranchExchangeInstruction
	toReturn.raw = raw
	toReturn.condition = getCondition(raw)
	toReturn.rn = NewARMRegister(uint8(raw & 0xf))
	return &toReturn, nil
}

func parseSingleDataSwapInstruction(raw uint32) (ARMInstruction, error) {
	var toReturn SingleDataSwapInstruction
	toReturn.raw = raw
	toReturn.condition = getCondition(raw)
	toReturn.rm = NewARMRegister(uint8(raw & 0xf))
	toReturn.rd = NewARMRegister(uint8((raw >> 12) & 0xf))
	toReturn.rn = NewARMRegister(uint8((raw >> 16) & 0xf))
	toReturn.byteQuantity = (raw & 0x400000) != 0
	return &toReturn, nil
}

func parseMultiplyInstruction(raw uint32) (ARMInstruction, error) {
	var toReturn MultiplyInstruction
	toReturn.raw = raw
	toReturn.condition = getCondition(raw)
	toReturn.isLongMultiply = (raw & 0x800000) != 0
	rm := uint8(raw & 0xf)
	rs := uint8((raw >> 8) & 0xf)
	rn := uint8((raw >> 12) & 0xf)
	rd := uint8((raw >> 16) & 0xf)
	if (rm == 15) || (rs == 15) || (rd == 15) {
		return nil, fmt.Errorf("Mutiply can't use r15")
	}
	if (rd == rm) || (rd == rs) {
		return nil, fmt.Errorf("Multiply destination and operand must differ.")
	}
	toReturn.rm = NewARMRegister(rm)
	toReturn.rs = NewARMRegister(rs)
	toReturn.rn = NewARMRegister(rn)
	toReturn.rd = NewARMRegister(rd)
	toReturn.setConditions = (raw & 0x100000) != 0
	toReturn.accumulate = (raw & 0x200000) != 0
	if toReturn.isLongMultiply || toReturn.accumulate {
		if rn == 15 {
			return nil, fmt.Errorf("Multiply can't use r15")
		}
		if rd == rn {
			return nil, fmt.Errorf("Multiply rd and rn must differ.")
		}
	}
	if toReturn.isLongMultiply {
		if rn == rm {
			return nil, fmt.Errorf("Invalid mull register combination.")
		}
		toReturn.signed = (raw & 0x400000) != 0
		toReturn.rdLow = toReturn.rn
		toReturn.rdHigh = toReturn.rd
	}
	return &toReturn, nil
}

func parsePSRTransferInstruction(raw uint32) (ARMInstruction, error) {
	var toReturn PSRTransferInstruction
	toReturn.raw = raw
	toReturn.condition = getCondition(raw)
	toReturn.useCPSR = (raw & 0x400000) == 0
	toReturn.writePSR = (raw & 0x200000) != 0
	if toReturn.writePSR {
		toReturn.rm = NewARMRegister(uint8(raw & 0xf))
		toReturn.flagsOnly = (raw & 0x10000) == 0
		if toReturn.flagsOnly {
			toReturn.isImmediate = (raw & 0x2000000) != 0
			if toReturn.isImmediate {
				toReturn.immediate = uint8(raw & 0xff)
				toReturn.rotate = uint8((raw >> 8) & 0xf)
			}
		}
	} else {
		toReturn.rd = NewARMRegister(uint8((raw >> 12) & 0xf))
	}
	return &toReturn, nil
}

func parseDataProcessingInstruction(raw uint32) (ARMInstruction, error) {
	var toReturn DataProcessingInstruction
	toReturn.raw = raw
	toReturn.setConditions = (raw & 0x100000) != 0
	if !toReturn.setConditions {
		if (raw & psrTransferMask) == psrTransferSet {
			return parsePSRTransferInstruction(raw)
		}
	}
	toReturn.condition = getCondition(raw)
	toReturn.isImmediate = (raw & 0x2000000) != 0
	if toReturn.isImmediate {
		toReturn.immediate = uint8(raw & 0xff)
		toReturn.rotate = uint8((raw >> 8) & 0xf)
	} else {
		toReturn.rm = NewARMRegister(uint8(raw & 0xf))
		toReturn.shift = NewARMShift(uint8((raw >> 4) & 0xff))
	}
	toReturn.rd = NewARMRegister(uint8((raw >> 12) & 0xf))
	toReturn.rn = NewARMRegister(uint8((raw >> 16) & 0xf))
	toReturn.opcode = NewARMDataProcessingOpcode(uint8((raw >> 21) & 0xf))
	return &toReturn, nil
}

func ParseInstruction(raw uint32) (ARMInstruction, error) {
	if (raw & 0x08000000) != 0 {
		if (raw & 0x04000000) != 0 {
			if (raw & 0x02000000) != 0 {
				if (raw & 0x01000000) != 0 {
					return parseSoftwareInterruptInstruction(raw)
				} else if (raw & 0x10) != 0 {
					return parseCoprocRegisterTransferInstruction(raw)
				}
				return parseCoprocDataOperationInstruction(raw)
			}
			return parseCoprocDataTransferInstruction(raw)
		}
		if (raw & 0x02000000) != 0 {
			return parseBranchInstruction(raw)
		}
		return parseBlockDataTransferInstruction(raw)
	}
	if (raw & 0x04000000) != 0 {
		if (raw & 0x06000010) == 0x06000010 {
			return parseUndefinedInstruction(raw)
		}
		return parseSingleDataTransferInstruction(raw)
	}
	if (raw & 0x0ffffff0) == 0x012fff10 {
		return parseBranchExchangeInstruction(raw)
	}
	if (raw & 0xf0) == 0x90 {
		if (raw & 0x0fb00f00) == 0x01000000 {
			return parseSingleDataSwapInstruction(raw)
		}
		if ((raw & 0x0fc00000) == 0) || ((raw & 0x0f800000) == 0x00800000) {
			return parseMultiplyInstruction(raw)
		}
	}
	if ((raw & 0x0e400f90) == 0x00000090) ||
		((raw & 0x0e400090) == 0x00400090) {
		return parseHalfwordDataTransferInstruction(raw)
	}
	return parseDataProcessingInstruction(raw)
}
