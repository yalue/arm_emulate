package arm_emulate

import (
	"fmt"
)

type THUMBInstruction interface {
	fmt.Stringer
	Raw() uint16
	Emulate(p ARMProcessor) error
}

type basicTHUMBInstruction struct {
	raw uint16
}

func (n *basicTHUMBInstruction) Raw() uint16 {
	return n.raw
}

func (n *basicTHUMBInstruction) String() string {
	return fmt.Sprintf("data: 0x%04x", n.raw)
}

func (n *basicTHUMBInstruction) Emulate(p ARMProcessor) error {
	return fmt.Errorf("Emulation not implemented for 0x%02x", n.raw)
}

type MoveShiftedRegisterInstruction struct {
	basicTHUMBInstruction
	rd        ARMRegister
	rs        ARMRegister
	offset    uint8
	operation uint8
}

func (n *MoveShiftedRegisterInstruction) String() string {
	var start string
	if n.operation == 0 {
		start = "lsl"
	} else if n.operation == 1 {
		start = "lsr"
	} else {
		start = "asr"
	}
	return fmt.Sprintf("%s %s, %s, %d", start, n.rd, n.rs, n.offset)
}

type AddSubtractInstruction struct {
	basicTHUMBInstruction
	isImmediate bool
	subtract    bool
	immediate   uint8
	rn          ARMRegister
	rs          ARMRegister
	rd          ARMRegister
}

func (n *AddSubtractInstruction) String() string {
	var start string
	if n.subtract {
		start = "sub"
	} else {
		start = "add"
	}
	start += fmt.Sprintf(" %s, %s, ", n.rd, n.rs)
	if n.isImmediate {
		start += fmt.Sprintf("%d", n.immediate)
	} else {
		start += n.rn.String()
	}
	return start
}

type MoveCompareAddSubtractImmediateInstruction struct {
	basicTHUMBInstruction
	rd        ARMRegister
	operation uint8
	immediate uint8
}

func (n *MoveCompareAddSubtractImmediateInstruction) String() string {
	var start string
	if n.operation == 0 {
		start = "mov"
	} else if n.operation == 1 {
		start = "cmp"
	} else if n.operation == 2 {
		start = "add"
	} else {
		start = "sub"
	}
	return fmt.Sprintf("%s %s, %d", start, n.rd, n.immediate)
}

type ALUOperationInstruction struct {
	basicTHUMBInstruction
	opcode ALUOpcodeTHUMB
	rd     ARMRegister
	rs     ARMRegister
}

func (n *ALUOperationInstruction) String() string {
	return fmt.Sprintf("%s %s, %s", n.opcode, n.rd, n.rs)
}

type HighRegisterOperationInstruction struct {
	basicTHUMBInstruction
	rd        ARMRegister
	rs        ARMRegister
	highFlag1 bool
	highFlag2 bool
	operation uint8
}

func (n *HighRegisterOperationInstruction) String() string {
	var start string
	if n.operation == 0 {
		start = "add"
	} else if n.operation == 1 {
		start = "cmp"
	} else if n.operation == 2 {
		start = "mov"
	} else {
		return fmt.Sprintf("bx %s", n.rs)
	}
	return fmt.Sprintf("%s %s, %s", start, n.rd, n.rs)
}

type PcRelativeLoadInstruction struct {
	basicTHUMBInstruction
	offset uint8
	rd     ARMRegister
}

func (n *PcRelativeLoadInstruction) String() string {
	return fmt.Sprintf("ldr %s, [pc, %d]", n.rd, uint16(n.offset)<<2)
}

type LoadStoreRegisterOffsetInstruction struct {
	basicTHUMBInstruction
	rd           ARMRegister
	rb           ARMRegister
	ro           ARMRegister
	byteQuantity bool
	load         bool
}

func (n *LoadStoreRegisterOffsetInstruction) String() string {
	var start string
	if n.load {
		start = "ldr"
	} else {
		start = "str"
	}
	if n.byteQuantity {
		start += "b"
	}
	return fmt.Sprintf("%s %s, [%s, %s]", start, n.rd, n.rb, n.ro)
}

type LoadStoreSignExtendedHalfwordInstruction struct {
	basicTHUMBInstruction
	rd         ARMRegister
	rb         ARMRegister
	ro         ARMRegister
	signExtend bool
	hBit       bool
}

func (n *LoadStoreSignExtendedHalfwordInstruction) String() string {
	var start string
	if n.signExtend {
		if n.hBit {
			start = "ldsh"
		} else {
			start = "ldsb"
		}
	} else {
		if n.hBit {
			start = "ldrh"
		} else {
			start = "strh"
		}
	}
	return fmt.Sprintf("%s %s, [%s, %s]", start, n.rd, n.rb, n.ro)
}

type LoadStoreImmediateOffsetInstruction struct {
	basicTHUMBInstruction
	rd           ARMRegister
	rb           ARMRegister
	offset       uint8
	load         bool
	byteQuantity bool
}

func (n *LoadStoreImmediateOffsetInstruction) String() string {
	var start string
	if n.load {
		start = "ldr"
	} else {
		start = "str"
	}
	offset := n.offset
	if n.byteQuantity {
		start += "b"
	} else {
		offset = offset << 2
	}
	return fmt.Sprintf("%s %s, [%s, %d]", start, n.rd, n.rb, offset)
}

type LoadStoreHalfwordInstruction struct {
	basicTHUMBInstruction
	rd     ARMRegister
	rb     ARMRegister
	offset uint8
	load   bool
}

func (n *LoadStoreHalfwordInstruction) String() string {
	var start string
	if n.load {
		start = "ldrh"
	} else {
		start = "strh"
	}
	return fmt.Sprintf("%s %s, [%s, %d]", start, n.rd, n.rb, n.offset<<1)
}

type SPRelativeLoadStoreInstruction struct {
	basicTHUMBInstruction
	offset uint8
	rd     ARMRegister
	load   bool
}

func (n *SPRelativeLoadStoreInstruction) String() string {
	var start string
	if n.load {
		start = "ldr"
	} else {
		start = "str"
	}
	return fmt.Sprintf("%s %s, [sp, %d]", start, n.rd, uint16(n.offset)<<1)
}

type LoadAddressInstruction struct {
	basicTHUMBInstruction
	offset uint8
	rd     ARMRegister
	loadSP bool
}

func (n *LoadAddressInstruction) String() string {
	var source string
	if n.loadSP {
		source = "sp"
	} else {
		source = "pc"
	}
	return fmt.Sprintf("add %s, %s, %d", n.rd, source, uint16(n.offset)<<2)
}

type AddToStackPointerInstruction struct {
	basicTHUMBInstruction
	offset   uint8
	negative bool
}

func (n *AddToStackPointerInstruction) String() string {
	offset := int(n.offset) << 2
	if n.negative {
		offset = -offset
	}
	return fmt.Sprintf("add sp, %d", offset)
}

type PushPopRegistersInstruction struct {
	basicTHUMBInstruction
	registerList  uint8
	storeLRLoadPC bool
	load          bool
}

// Like listString() for the block data transfer instruction, but doesn't
// include the curly braces.
func registerListStringTHUMB(bits uint8) string {
	toReturn := ""
	registers := bits
	consecutive := uint8(0)
	// As in the block data transfer (ARM) instruction, run for 9 iterations to
	// properly include r7 if it is set.
	for i := uint8(0); i < 9; i++ {
		if (registers & 1) == 1 {
			consecutive++
		} else if consecutive != 0 {
			startRegister := i - consecutive
			endRegister := i - 1
			consecutive = 0
			if len(toReturn) != 0 {
				toReturn += ", "
			}
			if startRegister == endRegister {
				toReturn += fmt.Sprintf("r%d", endRegister)
			} else {
				toReturn += fmt.Sprintf("r%d-r%d", startRegister, endRegister)
			}
		}
		registers = registers >> 1
	}
	return toReturn
}

func (n *PushPopRegistersInstruction) String() string {
	var start string
	registerList := registerListStringTHUMB(n.registerList)
	if n.load {
		start = "pop"
		if n.storeLRLoadPC {
			registerList += ", pc"
		}
	} else {
		start = "push"
		if n.storeLRLoadPC {
			registerList += ", lr"
		}
	}
	return fmt.Sprintf("%s {%s}", start, registerList)

}

type MultipleLoadStoreInstruction struct {
	basicTHUMBInstruction
	registerList uint8
	rb           ARMRegister
	load         bool
}

func (n *MultipleLoadStoreInstruction) String() string {
	var start string
	if n.load {
		start = "ldmia"
	} else {
		start = "stmia"
	}
	registers := registerListStringTHUMB(n.registerList)
	return fmt.Sprintf("%s %s!, {%s}", start, n.rb, registers)
}

type ConditionalBranchInstruction struct {
	basicTHUMBInstruction
	offset    uint8
	condition ARMCondition
}

func (n *ConditionalBranchInstruction) String() string {
	// Offset must be a signed type before it is converted to 16-bits
	offset := int16(int8(n.offset)) << 1
	return fmt.Sprintf("b%s %d", n.condition, offset)
}

type SoftwareInterruptTHUMBInstruction struct {
	basicTHUMBInstruction
	comment uint8
}

func (n *SoftwareInterruptTHUMBInstruction) String() string {
	return fmt.Sprintf("swi %d", n.comment)
}

type unConditionalBranchInstruction struct {
	basicTHUMBInstruction
	offset uint16
}

func (n *unConditionalBranchInstruction) String() string {
	// Take care of sign extending and left-shifting by 1
	offset := int16(n.offset<<5) >> 4
	return fmt.Sprintf("b %d", offset)
}

type LongBranchAndLinkInstruction struct {
	basicTHUMBInstruction
	offset    uint16
	offsetLow bool
}

func (n *LongBranchAndLinkInstruction) String() string {
	if n.offsetLow {
		return fmt.Sprintf("bl lr + %d (long branch and link)", n.offset<<1)
	}
	return fmt.Sprintf("add lr, pc, %d (long branch and link)",
		(int32(n.offset)<<21)>>9)
}

func parseMoveShiftedRegisterInstruction(r uint16) (THUMBInstruction, error) {
	var toReturn MoveShiftedRegisterInstruction
	toReturn.raw = r
	toReturn.rd = NewARMRegister(uint8(r & 7))
	toReturn.rs = NewARMRegister(uint8((r >> 3) & 7))
	toReturn.offset = uint8((r >> 6) & 0x1f)
	toReturn.operation = uint8((r >> 11) & 3)
	return &toReturn, nil
}

func parseAddSubtractInstruction(raw uint16) (THUMBInstruction, error) {
	var toReturn AddSubtractInstruction
	toReturn.raw = raw
	toReturn.rd = NewARMRegister(uint8(raw & 7))
	toReturn.rs = NewARMRegister(uint8((raw >> 3) & 7))
	toReturn.subtract = (raw & 0x200) != 0
	toReturn.immediate = uint8((raw >> 6) & 7)
	toReturn.isImmediate = (raw & 0x400) != 0
	if !toReturn.isImmediate {
		toReturn.rn = NewARMRegister(toReturn.immediate)
	}
	return &toReturn, nil
}

func parseMoveCompareAddSubtractImmediateInstruction(raw uint16) (
	THUMBInstruction, error) {
	var toReturn MoveCompareAddSubtractImmediateInstruction
	toReturn.raw = raw
	toReturn.immediate = uint8(raw & 0xff)
	toReturn.operation = uint8((raw >> 11) & 3)
	toReturn.rd = NewARMRegister(uint8((raw >> 8) & 7))
	return &toReturn, nil
}

func parseALUOperationInstruction(raw uint16) (THUMBInstruction, error) {
	var toReturn ALUOperationInstruction
	toReturn.raw = raw
	toReturn.opcode = NewALUOpcodeTHUMB(uint8((raw >> 6) & 0xf))
	toReturn.rd = NewARMRegister(uint8(raw & 7))
	toReturn.rs = NewARMRegister(uint8((raw >> 3) & 7))
	return &toReturn, nil
}

func parseHighRegisterOperationInstruction(raw uint16) (THUMBInstruction,
	error) {
	var toReturn HighRegisterOperationInstruction
	var h bool
	var register uint8
	toReturn.raw = raw
	toReturn.operation = uint8((raw >> 8) & 3)
	h = (raw & 0x80) != 0
	toReturn.highFlag1 = h
	register = uint8(raw & 7)
	if h {
		register += 8
	}
	toReturn.rd = NewARMRegister(register)
	h = (raw & 0x40) != 0
	toReturn.highFlag2 = h
	register = uint8((raw >> 3) & 7)
	if h {
		register += 8
	}
	toReturn.rs = NewARMRegister(register)
	return &toReturn, nil
}

func parsePCRelativeLoadInstruction(raw uint16) (THUMBInstruction, error) {
	var toReturn PcRelativeLoadInstruction
	toReturn.raw = raw
	toReturn.offset = uint8(raw & 0xff)
	toReturn.rd = NewARMRegister(uint8((raw >> 8) & 7))
	return &toReturn, nil
}

func parseLoadStoreRegisterOffsetInstruction(raw uint16) (THUMBInstruction,
	error) {
	var toReturn LoadStoreRegisterOffsetInstruction
	toReturn.raw = raw
	toReturn.byteQuantity = (raw & 0x400) != 0
	toReturn.load = (raw & 0x800) != 0
	toReturn.rd = NewARMRegister(uint8(raw & 7))
	toReturn.rb = NewARMRegister(uint8((raw >> 3) & 7))
	toReturn.ro = NewARMRegister(uint8((raw >> 6) & 7))
	return &toReturn, nil
}

func parseLoadStoreSignExtendedHalfwordInstruction(raw uint16) (
	THUMBInstruction, error) {
	var toReturn LoadStoreSignExtendedHalfwordInstruction
	toReturn.raw = raw
	toReturn.rd = NewARMRegister(uint8(raw & 7))
	toReturn.rb = NewARMRegister(uint8((raw >> 3) & 7))
	toReturn.ro = NewARMRegister(uint8((raw >> 6) & 7))
	toReturn.signExtend = (raw & 0x400) != 0
	toReturn.hBit = (raw & 0x800) != 0
	return &toReturn, nil
}

func parseLoadStoreImmediateOffsetInstruction(raw uint16) (THUMBInstruction,
	error) {
	var toReturn LoadStoreImmediateOffsetInstruction
	toReturn.raw = raw
	toReturn.rd = NewARMRegister(uint8(raw & 7))
	toReturn.rb = NewARMRegister(uint8((raw >> 3) & 7))
	toReturn.offset = uint8((raw >> 6) & 0x1f)
	toReturn.load = (raw & 0x800) != 0
	toReturn.byteQuantity = (raw & 0x1000) != 0
	return &toReturn, nil
}

func parseLoadStoreHalfwordInstruction(raw uint16) (THUMBInstruction, error) {
	var toReturn LoadStoreHalfwordInstruction
	toReturn.raw = raw
	toReturn.rd = NewARMRegister(uint8(raw & 7))
	toReturn.rb = NewARMRegister(uint8((raw >> 3) & 7))
	toReturn.offset = uint8((raw >> 6) & 0x1f)
	toReturn.load = (raw & 0x800) != 0
	return &toReturn, nil
}

func parseSPRelativeLoadStoreInstruction(raw uint16) (THUMBInstruction,
	error) {
	var toReturn SPRelativeLoadStoreInstruction
	toReturn.raw = raw
	toReturn.offset = uint8(raw & 0xff)
	toReturn.rd = NewARMRegister(uint8((raw >> 8) & 0x7))
	toReturn.load = (raw & 0x800) != 0
	return &toReturn, nil
}

func parseLoadAddressInstruction(raw uint16) (THUMBInstruction, error) {
	var toReturn LoadAddressInstruction
	toReturn.raw = raw
	toReturn.offset = uint8(raw & 0xff)
	toReturn.rd = NewARMRegister(uint8((raw >> 8) & 0x7))
	toReturn.loadSP = (raw & 0x800) != 0
	return &toReturn, nil
}

func parseAddToStackPointerInstruction(raw uint16) (THUMBInstruction, error) {
	var toReturn AddToStackPointerInstruction
	toReturn.raw = raw
	toReturn.offset = uint8(raw & 0x7f)
	toReturn.negative = (raw & 0x80) != 0
	return &toReturn, nil
}

func parsePushPopRegistersInstruction(raw uint16) (THUMBInstruction, error) {
	var toReturn PushPopRegistersInstruction
	toReturn.raw = raw
	toReturn.registerList = uint8(raw & 0xff)
	toReturn.storeLRLoadPC = (raw & 0x100) != 0
	toReturn.load = (raw & 0x800) != 0
	return &toReturn, nil
}

func parseMultipleLoadStoreInstruction(raw uint16) (THUMBInstruction, error) {
	var toReturn MultipleLoadStoreInstruction
	toReturn.raw = raw
	toReturn.registerList = uint8(raw & 0xff)
	toReturn.rb = NewARMRegister(uint8((raw >> 8) & 0x7))
	toReturn.load = (raw & 0x800) != 0
	return &toReturn, nil
}

func parseConditionalBranchInstruction(raw uint16) (THUMBInstruction, error) {
	var toReturn ConditionalBranchInstruction
	toReturn.raw = raw
	toReturn.offset = uint8(raw & 0xff)
	toReturn.condition = NewARMCondition(uint8((raw >> 8) & 0xf))
	if toReturn.condition.Condition() == 14 {
		return &toReturn, fmt.Errorf("Illegal condition in conditional branch")
	}
	return &toReturn, nil
}

func parseSoftwareInterruptTHUMBInstruction(raw uint16) (THUMBInstruction,
	error) {
	var toReturn SoftwareInterruptTHUMBInstruction
	toReturn.raw = raw
	toReturn.comment = uint8(raw & 0xff)
	return &toReturn, nil
}

func parseUnConditionalBranchInstruction(raw uint16) (THUMBInstruction,
	error) {
	var toReturn unConditionalBranchInstruction
	toReturn.raw = raw
	toReturn.offset = raw & 0x7ff
	return &toReturn, nil
}

func parseLongBranchAndLinkInstruction(raw uint16) (THUMBInstruction, error) {
	var toReturn LongBranchAndLinkInstruction
	toReturn.raw = raw
	toReturn.offset = raw & 0x7ff
	toReturn.offsetLow = (raw & 0x800) != 0
	return &toReturn, nil
}

func ParseTHUMBInstruction(raw uint16) (THUMBInstruction, error) {
	if (raw & 0x8000) != 0 {
		if (raw & 0x4000) != 0 {
			if (raw & 0x2000) != 0 {
				if (raw & 0x1000) != 0 {
					return parseLongBranchAndLinkInstruction(raw)
				}
				return parseUnConditionalBranchInstruction(raw)
			}
			if (raw & 0x3000) != 0 {
				if (raw & 0x0f00) == 0x0f00 {
					return parseSoftwareInterruptTHUMBInstruction(raw)
				}
				return parseConditionalBranchInstruction(raw)
			}
			return parseMultipleLoadStoreInstruction(raw)
		}
		if (raw & 0x3000) != 0x3000 {
			if (raw & 0x1000) == 0 {
				if (raw & 0x2000) != 0 {
					return parseLoadAddressInstruction(raw)
				}
				return parseLoadStoreHalfwordInstruction(raw)
			}
			return parseSPRelativeLoadStoreInstruction(raw)
		}
		if (raw & 0x0f00) != 0 {
			return parsePushPopRegistersInstruction(raw)
		}
		return parseAddToStackPointerInstruction(raw)
	}
	if (raw & 0x4000) == 0 {
		if (raw & 0x2000) == 0 {
			if (raw & 0x1800) == 0x1800 {
				return parseAddSubtractInstruction(raw)
			}
			return parseMoveShiftedRegisterInstruction(raw)
		}
		return parseMoveCompareAddSubtractImmediateInstruction(raw)
	}
	if (raw & 0x2000) == 0 {
		if (raw & 0x1000) == 0 {
			if (raw & 0x800) == 0 {
				if (raw & 0x400) == 0 {
					return parseALUOperationInstruction(raw)
				}
				return parseHighRegisterOperationInstruction(raw)
			}
			return parsePCRelativeLoadInstruction(raw)
		}
		if (raw & 0x200) == 0 {
			return parseLoadStoreRegisterOffsetInstruction(raw)
		} else {
			return parseLoadStoreSignExtendedHalfwordInstruction(raw)
		}
	}
	return parseLoadStoreImmediateOffsetInstruction(raw)
}
