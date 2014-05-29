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

// This is the main interface which all 32-bit ARM instructions support.
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
	Opcode        ARMDataProcessingOpcode
	Rm            ARMRegister
	Shift         ARMShift
	Immediate     uint8
	Rotate        uint8
	Rd            ARMRegister
	Rn            ARMRegister
	SetConditions bool
	IsImmediate   bool
}

func (n *DataProcessingInstruction) secondOperand() string {
	if n.IsImmediate {
		r := n.Rotate << 1
		value := uint32(n.Immediate)
		value = (value >> r) | (value << (32 - r))
		return fmt.Sprintf("%d", value)
	}
	toReturn := n.Rm.String()
	if n.Shift.UseRegister() || (n.Shift.Amount() != 0) {
		toReturn += " "
	}
	return fmt.Sprintf("%s%s", toReturn, n.Shift)
}

func (n *DataProcessingInstruction) String() string {
	prefix := n.Opcode.String()
	prefix += n.condition.String()
	opcodeValue := n.Opcode.Value()
	switch opcodeValue {
	case movARMOpcode, mvnARMOpcode:
		if n.SetConditions {
			prefix += "s"
		}
		return fmt.Sprintf("%s %s, %s", prefix, n.Rd, n.secondOperand())
	case tstARMOpcode, teqARMOpcode, cmnARMOpcode, cmpARMOpcode:
		return fmt.Sprintf("%s %s, %s", prefix, n.Rn, n.secondOperand())
	}
	if n.SetConditions {
		prefix += "s"
	}
	return fmt.Sprintf("%s %s, %s, %s", prefix, n.Rd, n.Rn, n.secondOperand())
}

type PSRTransferInstruction struct {
	basicARMInstruction
	Rm          ARMRegister
	Rd          ARMRegister
	IsImmediate bool
	WritePSR    bool
	UseCPSR     bool
	FlagsOnly   bool
	Immediate   uint8
	Rotate      uint8
}

func (n *PSRTransferInstruction) String() string {
	var usedPSR string
	if n.UseCPSR {
		usedPSR = "cpsr"
	} else {
		usedPSR = "spsr"
	}
	if !n.WritePSR {
		return fmt.Sprintf("mrs%s %s, %s", n.condition, n.Rd, usedPSR)
	}
	if !n.FlagsOnly {
		return fmt.Sprintf("msr%s %s, %s", n.condition, n.Rm, usedPSR)
	}
	usedPSR += "_flags"
	if n.IsImmediate {
		r := n.Rotate << 1
		value := uint32(n.Immediate)
		value = (value >> r) | (value << (32 - r))
		return fmt.Sprintf("msr%s %s, %d", n.condition, usedPSR, value)
	}
	return fmt.Sprintf("msr%s %s, %s", n.condition, usedPSR, n.Rm)
}

// This includes both multiply and long multiply instructions
type MultiplyInstruction struct {
	basicARMInstruction
	IsLongMultiply bool
	Rm             ARMRegister
	Rn             ARMRegister
	Rs             ARMRegister
	Rd             ARMRegister
	RdLow          ARMRegister
	RdHigh         ARMRegister
	SetConditions  bool
	Accumulate     bool
	Signed         bool
}

func (n *MultiplyInstruction) String() string {
	var start string
	if n.Accumulate {
		start = "mla"
	} else {
		start = "mul"
	}
	if n.IsLongMultiply {
		if n.Signed {
			start = "s" + start + "l"
		} else {
			start = "u" + start + "l"
		}
	}
	start += n.condition.String()
	if n.SetConditions {
		start += "s"
	}
	if n.IsLongMultiply {
		return fmt.Sprintf("%s %s, %s, %s, %s", start, n.RdLow, n.RdHigh, n.Rm,
			n.Rs)
	}
	if !n.Accumulate {
		return fmt.Sprintf("%s %s, %s, %s", start, n.Rd, n.Rm, n.Rs)
	}
	return fmt.Sprintf("%s %s, %s, %s, %s", start, n.Rd, n.Rm, n.Rs, n.Rn)
}

type SingleDataSwapInstruction struct {
	basicARMInstruction
	Rm           ARMRegister
	Rn           ARMRegister
	Rd           ARMRegister
	ByteQuantity bool
}

func (n *SingleDataSwapInstruction) String() string {
	start := "swp"
	start += n.condition.String()
	if n.ByteQuantity {
		start += "b"
	}
	return fmt.Sprintf("%s %s, %s, [%s]", start, n.Rd, n.Rm, n.Rn)
}

type BranchExchangeInstruction struct {
	basicARMInstruction
	Rn ARMRegister
}

func (n *BranchExchangeInstruction) String() string {
	return fmt.Sprintf("bx%s %s", n.condition, n.Rn)
}

type HalfwordDataTransferInstruction struct {
	basicARMInstruction
	IsImmediate bool
	Halfword    bool
	Signed      bool
	Rm          ARMRegister
	Rn          ARMRegister
	Rd          ARMRegister
	Offset      uint8
	Load        bool
	WriteBack   bool
	Up          bool
	Preindex    bool
}

func (n *HalfwordDataTransferInstruction) String() string {
	var start string
	if n.Load {
		start = "ldr"
	} else {
		start = "str"
	}
	start += n.condition.String()
	if n.Signed {
		start += "s"
	}
	if n.Halfword {
		start += "h"
	} else {
		start += "b"
	}
	start += " " + n.Rd.String() + ","
	offset := int(n.Offset)
	offsetReg := n.Rm.String()
	if n.Rn.Register() == 15 {
		offset += 8
	}
	if !n.Up {
		offset = -offset
		offsetReg = "-" + offsetReg
	}
	if n.IsImmediate && n.Preindex && !n.WriteBack && (n.Rn.Register() == 15) {
		return fmt.Sprintf("%s %d", start, offset)
	}
	if n.Preindex {
		postfix := ""
		if n.WriteBack {
			postfix = "!"
		}
		if n.IsImmediate {
			if n.Offset == 0 {
				return fmt.Sprintf("%s [%s]%s", start, n.Rn, postfix)
			}
			return fmt.Sprintf("%s [%s, %d]%s", start, n.Rn, offset, postfix)
		}
		return fmt.Sprintf("%s [%s, %s]%s", start, n.Rn, offsetReg, postfix)
	}
	if n.IsImmediate {
		return fmt.Sprintf("%s [%s], %d", start, n.Rn, offset)
	}
	return fmt.Sprintf("%s [%s], %s", start, n.Rn, offsetReg)
}

type SingleDataTransferInstruction struct {
	basicARMInstruction
	Rn              ARMRegister
	Rd              ARMRegister
	Rm              ARMRegister
	Shift           ARMShift
	Offset          uint16
	Load            bool
	WriteBack       bool
	ByteQuantity    bool
	Up              bool
	Preindex        bool
	ImmediateOffset bool
}

func (n *SingleDataTransferInstruction) String() string {
	var start string
	if n.Load {
		start = "ldr"
	} else {
		start = "str"
	}
	start += n.condition.String()
	if n.ByteQuantity {
		start += "b"
	}
	if !n.Preindex && n.WriteBack {
		start += "t"
	}
	start += " " + n.Rd.String() + ","
	upString := ""
	if !n.Up {
		upString = "-"
	}
	shiftString := ""
	if !n.ImmediateOffset && (n.Shift.Amount() != 0) {
		shiftString = ", " + n.Shift.String()
	}
	offset := int(n.Offset)
	if n.Rn.Register() == 15 {
		offset += 8
	}
	if n.Preindex {
		postfix := ""
		if n.WriteBack {
			postfix = "!"
		}
		if n.ImmediateOffset {
			if (n.Rn.Register() == 15) && (offset != 0) {
				return fmt.Sprintf("%s %s%d", start, upString, offset)
			}
			if offset == 0 {
				return fmt.Sprintf("%s [%s]%s", start, n.Rn, postfix)
			}
			return fmt.Sprintf("%s [%s, %s%d]%s", start, n.Rn, upString,
				offset, postfix)
		}
		return fmt.Sprintf("%s [%s, %s%s%s]%s", start, n.Rn, upString, n.Rm,
			shiftString, postfix)
	}
	if n.ImmediateOffset {
		return fmt.Sprintf("%s [%s], %s%d", start, n.Rn, upString, offset)
	}
	return fmt.Sprintf("%s [%s], %s%s%s", start, n.Rn, upString, n.Rm,
		shiftString)
}

type UndefinedInstruction struct {
	basicARMInstruction
}

type BlockDataTransferInstruction struct {
	basicARMInstruction
	RegisterList uint16
	Rn           ARMRegister
	Load         bool
	WriteBack    bool
	ForceUser    bool
	Up           bool
	Preindex     bool
}

func (n *BlockDataTransferInstruction) listString() string {
	var s string
	consecutive := uint8(0)
	s = "{"
	registers := n.RegisterList
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
	if n.Load {
		start = "ldm"
	} else {
		start = "stm"
	}
	// The mnemonic postfix depends on the u and p bits and stack usage
	if n.Rn.Register() == 13 {
		if n.Up {
			if n.Preindex {
				start += "ed"
			} else {
				start += "fd"
			}
		} else {
			if n.Preindex {
				start += "ea"
			} else {
				start += "fa"
			}
		}
	} else {
		if n.Up {
			if n.Preindex {
				start += "ib"
			} else {
				start += "ia"
			}
		} else {
			if n.Preindex {
				start += "db"
			} else {
				start += "da"
			}
		}
	}
	start += " " + n.Rn.String()
	if n.WriteBack {
		start += "!"
	}
	start += ", " + n.listString()
	if n.ForceUser {
		start += "^"
	}
	return start
}

type BranchInstruction struct {
	basicARMInstruction
	Offset int32
	Link   bool
}

func (n *BranchInstruction) String() string {
	start := "b"
	if n.Link {
		start += "l"
	}
	start += n.condition.String()
	// Sign extend and shift right by 2 bits...
	offset := n.Offset << 8
	offset = offset >> 6
	return fmt.Sprintf("%s %d", start, offset)
}

type CoprocDataTransferInstruction struct {
	basicARMInstruction
	Rn           ARMRegister
	CoprocNumber uint8
	CoprocRd     uint8
	Offset       uint8
	Load         bool
	WriteBack    bool
	LongTransfer bool
	Up           bool
	Preindex     bool
}

func (n *CoprocDataTransferInstruction) String() string {
	var start string
	if n.Load {
		start = "ldc"
	} else {
		start = "stc"
	}
	start += n.condition.String()
	if n.LongTransfer {
		start += "l"
	}
	start += fmt.Sprintf(" p%d, c%d,", n.CoprocNumber, n.CoprocRd)
	offset := int(n.Offset) << 2
	if !n.Up {
		offset = -offset
	}
	if n.Rn.Register() == 15 {
		offset += 8
	}
	if n.Preindex {
		postfix := ""
		if n.WriteBack {
			postfix = "!"
		}
		if !n.WriteBack && (n.Rn.Register() == 15) {
			return fmt.Sprintf("%s %d", start, offset)
		}
		if n.Offset == 0 {
			return fmt.Sprintf("%s [%s]%s", start, n.Rn, postfix)
		}
		return fmt.Sprintf("%s [%s, %d]%s", start, n.Rn, offset, postfix)
	}
	if n.Offset == 0 {
		return fmt.Sprintf("%s %s", start, n.Rn)
	}
	return fmt.Sprintf("%s [%s], %d", start, n.Rn, offset)
}

type CoprocDataOperationInstruction struct {
	basicARMInstruction
	CoprocNumber uint8
	CoprocOpcode uint8
	CoprocInfo   uint8
	CoprocRn     uint8
	CoprocRd     uint8
	CoprocRm     uint8
}

func (n *CoprocDataOperationInstruction) String() string {
	return fmt.Sprintf("cdp%s p%d, %d, c%d, c%d, c%d, %d", n.condition,
		n.CoprocNumber, n.CoprocOpcode, n.CoprocRd, n.CoprocRn, n.CoprocRm,
		n.CoprocInfo)
}

type CoprocRegisterTransferInstruction struct {
	basicARMInstruction
	Rd            ARMRegister
	Load          bool
	CoprocNumber  uint8
	CoprocOpcode  uint8
	CoprocOperand uint8
	CoprocRn      uint8
	CoprocRm      uint8
}

func (n *CoprocRegisterTransferInstruction) String() string {
	var start string
	if n.Load {
		start = "mrc"
	} else {
		start = "mcr"
	}
	start += n.condition.String()
	return fmt.Sprintf("%s p%d, %d, %s, c%d, c%d, %d", start, n.CoprocNumber,
		n.CoprocOpcode, n.Rd, n.CoprocRn, n.CoprocRm, n.CoprocOperand)
}

type SoftwareInterruptInstruction struct {
	basicARMInstruction
	Comment uint32
}

func (n *SoftwareInterruptInstruction) String() string {
	return fmt.Sprintf("swi%s %08x", n.condition, n.Comment)
}

func getCondition(raw uint32) ARMCondition {
	return NewARMCondition(uint8((raw >> 28) & 0xf))
}

func parseSoftwareInterruptInstruction(raw uint32) (ARMInstruction, error) {
	var toReturn SoftwareInterruptInstruction
	toReturn.raw = raw
	toReturn.condition = getCondition(raw)
	toReturn.Comment = raw & 0x00ffffff
	return &toReturn, nil
}

func parseCoprocRegisterTransferInstruction(raw uint32) (ARMInstruction,
	error) {
	var toReturn CoprocRegisterTransferInstruction
	toReturn.raw = raw
	toReturn.condition = getCondition(raw)
	toReturn.Rd = NewARMRegister(uint8((raw >> 12) & 0xf))
	toReturn.Load = (raw & 0x100000) != 0
	toReturn.CoprocNumber = uint8((raw >> 8) & 0xf)
	toReturn.CoprocOpcode = uint8((raw >> 21) & 0x7)
	toReturn.CoprocOperand = uint8((raw >> 5) & 0x7)
	toReturn.CoprocRm = uint8(raw & 0xf)
	toReturn.CoprocRn = uint8((raw >> 16) & 0xf)
	return &toReturn, nil
}

func parseCoprocDataOperationInstruction(raw uint32) (ARMInstruction, error) {
	var toReturn CoprocDataOperationInstruction
	toReturn.raw = raw
	toReturn.condition = getCondition(raw)
	toReturn.CoprocNumber = uint8((raw >> 8) & 0xf)
	toReturn.CoprocRm = uint8(raw & 0xf)
	toReturn.CoprocRd = uint8((raw >> 12) & 0xf)
	toReturn.CoprocRn = uint8((raw >> 16) & 0xf)
	toReturn.CoprocOpcode = uint8((raw >> 20) & 0xf)
	toReturn.CoprocInfo = uint8((raw >> 5) & 0x7)
	return &toReturn, nil
}

func parseCoprocDataTransferInstruction(raw uint32) (ARMInstruction, error) {
	var toReturn CoprocDataTransferInstruction
	toReturn.raw = raw
	toReturn.condition = getCondition(raw)
	toReturn.Rn = NewARMRegister(uint8((raw >> 16) & 0xf))
	toReturn.Offset = uint8(raw & 0xff)
	toReturn.CoprocNumber = uint8((raw >> 8) & 0xf)
	toReturn.CoprocRd = uint8((raw >> 12) & 0xf)
	toReturn.Load = (raw & 0x100000) != 0
	toReturn.WriteBack = (raw & 0x200000) != 0
	toReturn.LongTransfer = (raw & 0x400000) != 0
	toReturn.Up = (raw & 0x800000) != 0
	toReturn.Preindex = (raw & 0x1000000) != 0
	return &toReturn, nil
}

func parseBranchInstruction(raw uint32) (ARMInstruction, error) {
	var toReturn BranchInstruction
	toReturn.raw = raw
	toReturn.condition = getCondition(raw)
	toReturn.Offset = int32(raw) & int32(0x00ffffff)
	toReturn.Link = (raw & 0x1000000) != 0
	return &toReturn, nil
}

func parseBlockDataTransferInstruction(raw uint32) (ARMInstruction, error) {
	var toReturn BlockDataTransferInstruction
	toReturn.raw = raw
	toReturn.condition = getCondition(raw)
	toReturn.RegisterList = uint16(raw & 0xffff)
	toReturn.Rn = NewARMRegister(uint8((raw >> 16) & 0xf))
	toReturn.Load = (raw & 0x100000) != 0
	toReturn.WriteBack = (raw & 0x200000) != 0
	toReturn.ForceUser = (raw & 0x400000) != 0
	toReturn.Up = (raw & 0x800000) != 0
	toReturn.Preindex = (raw & 0x1000000) != 0
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
	toReturn.ImmediateOffset = (raw & 0x2000000) == 0
	if !toReturn.ImmediateOffset {
		toReturn.Shift = NewARMShift(uint8((raw >> 4) & 0xff))
		// This shouldn't happen as along as the undefined instruction mask is
		// checked before the single data transfer instruction mask
		if toReturn.Shift.UseRegister() {
			var errorInstruction UndefinedInstruction
			errorInstruction.raw = raw
			errorInstruction.condition = toReturn.condition
			return &errorInstruction, fmt.Errorf("Illegal shift")
		}
		toReturn.Rm = NewARMRegister(uint8(raw & 0xf))
	} else {
		toReturn.Offset = uint16(raw & 0xfff)
	}
	toReturn.Rd = NewARMRegister(uint8((raw >> 12) & 0xf))
	toReturn.Rn = NewARMRegister(uint8((raw >> 16) & 0xf))
	toReturn.Load = (raw & 0x100000) != 0
	toReturn.WriteBack = (raw & 0x200000) != 0
	toReturn.ByteQuantity = (raw & 0x400000) != 0
	toReturn.Up = (raw & 0x800000) != 0
	toReturn.Preindex = (raw & 0x1000000) != 0
	return &toReturn, nil
}

func parseHalfwordDataTransferInstruction(raw uint32) (ARMInstruction, error) {
	var toReturn HalfwordDataTransferInstruction
	toReturn.raw = raw
	toReturn.condition = getCondition(raw)
	toReturn.IsImmediate = (raw & 0x400000) != 0
	if toReturn.IsImmediate {
		toReturn.Offset = uint8((raw & 0xf) | ((raw >> 4) & 0xf0))
	} else {
		toReturn.Rm = NewARMRegister(uint8(raw & 0xf))
	}
	toReturn.Halfword = (raw & 0x20) != 0
	toReturn.Signed = (raw & 0x40) != 0
	toReturn.Rd = NewARMRegister(uint8((raw >> 12) & 0xf))
	toReturn.Rn = NewARMRegister(uint8((raw >> 16) & 0xf))
	toReturn.Load = (raw & 0x100000) != 0
	toReturn.WriteBack = (raw & 0x200000) != 0
	toReturn.Up = (raw & 0x800000) != 0
	toReturn.Preindex = (raw & 0x1000000) != 0
	return &toReturn, nil
}

func parseBranchExchangeInstruction(raw uint32) (ARMInstruction, error) {
	var toReturn BranchExchangeInstruction
	toReturn.raw = raw
	toReturn.condition = getCondition(raw)
	toReturn.Rn = NewARMRegister(uint8(raw & 0xf))
	return &toReturn, nil
}

func parseSingleDataSwapInstruction(raw uint32) (ARMInstruction, error) {
	var toReturn SingleDataSwapInstruction
	toReturn.raw = raw
	toReturn.condition = getCondition(raw)
	toReturn.Rm = NewARMRegister(uint8(raw & 0xf))
	toReturn.Rd = NewARMRegister(uint8((raw >> 12) & 0xf))
	toReturn.Rn = NewARMRegister(uint8((raw >> 16) & 0xf))
	toReturn.ByteQuantity = (raw & 0x400000) != 0
	return &toReturn, nil
}

func parseMultiplyInstruction(raw uint32) (ARMInstruction, error) {
	var toReturn MultiplyInstruction
	toReturn.raw = raw
	toReturn.condition = getCondition(raw)
	toReturn.IsLongMultiply = (raw & 0x800000) != 0
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
	toReturn.Rm = NewARMRegister(rm)
	toReturn.Rs = NewARMRegister(rs)
	toReturn.Rn = NewARMRegister(rn)
	toReturn.Rd = NewARMRegister(rd)
	toReturn.SetConditions = (raw & 0x100000) != 0
	toReturn.Accumulate = (raw & 0x200000) != 0
	if toReturn.IsLongMultiply || toReturn.Accumulate {
		if rn == 15 {
			return nil, fmt.Errorf("Multiply can't use r15")
		}
		if rd == rn {
			return nil, fmt.Errorf("Multiply rd and rn must differ.")
		}
	}
	if toReturn.IsLongMultiply {
		if rn == rm {
			return nil, fmt.Errorf("Invalid mull register combination.")
		}
		toReturn.Signed = (raw & 0x400000) != 0
		toReturn.RdLow = toReturn.Rn
		toReturn.RdHigh = toReturn.Rd
	}
	return &toReturn, nil
}

func parsePSRTransferInstruction(raw uint32) (ARMInstruction, error) {
	var toReturn PSRTransferInstruction
	toReturn.raw = raw
	toReturn.condition = getCondition(raw)
	toReturn.UseCPSR = (raw & 0x400000) == 0
	toReturn.WritePSR = (raw & 0x200000) != 0
	if toReturn.WritePSR {
		toReturn.Rm = NewARMRegister(uint8(raw & 0xf))
		toReturn.FlagsOnly = (raw & 0x10000) == 0
		if toReturn.FlagsOnly {
			toReturn.IsImmediate = (raw & 0x2000000) != 0
			if toReturn.IsImmediate {
				toReturn.Immediate = uint8(raw & 0xff)
				toReturn.Rotate = uint8((raw >> 8) & 0xf)
			}
		}
	} else {
		toReturn.Rd = NewARMRegister(uint8((raw >> 12) & 0xf))
	}
	return &toReturn, nil
}

func parseDataProcessingInstruction(raw uint32) (ARMInstruction, error) {
	var toReturn DataProcessingInstruction
	toReturn.raw = raw
	toReturn.SetConditions = (raw & 0x100000) != 0
	if !toReturn.SetConditions {
		if (raw & psrTransferMask) == psrTransferSet {
			return parsePSRTransferInstruction(raw)
		}
	}
	toReturn.condition = getCondition(raw)
	toReturn.IsImmediate = (raw & 0x2000000) != 0
	if toReturn.IsImmediate {
		toReturn.Immediate = uint8(raw & 0xff)
		toReturn.Rotate = uint8((raw >> 8) & 0xf)
	} else {
		toReturn.Rm = NewARMRegister(uint8(raw & 0xf))
		toReturn.Shift = NewARMShift(uint8((raw >> 4) & 0xff))
	}
	toReturn.Rd = NewARMRegister(uint8((raw >> 12) & 0xf))
	toReturn.Rn = NewARMRegister(uint8((raw >> 16) & 0xf))
	toReturn.Opcode = NewARMDataProcessingOpcode(uint8((raw >> 21) & 0xf))
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
