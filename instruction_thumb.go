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
	Rd        ARMRegister
	Rs        ARMRegister
	Offset    uint8
	Operation uint8
}

func (n *MoveShiftedRegisterInstruction) String() string {
	var start string
	if n.Operation == 0 {
		start = "lsl"
	} else if n.Operation == 1 {
		start = "lsr"
	} else {
		start = "asr"
	}
	return fmt.Sprintf("%s %s, %s, %d", start, n.Rd, n.Rs, n.Offset)
}

type AddSubtractInstruction struct {
	basicTHUMBInstruction
	IsImmediate bool
	Subtract    bool
	Immediate   uint8
	Rn          ARMRegister
	Rs          ARMRegister
	Rd          ARMRegister
}

func (n *AddSubtractInstruction) String() string {
	var start string
	if n.Subtract {
		start = "sub"
	} else {
		start = "add"
	}
	start += fmt.Sprintf(" %s, %s, ", n.Rd, n.Rs)
	if n.IsImmediate {
		start += fmt.Sprintf("%d", n.Immediate)
	} else {
		start += n.Rn.String()
	}
	return start
}

type MoveCompareAddSubtractImmediateInstruction struct {
	basicTHUMBInstruction
	Rd        ARMRegister
	Operation uint8
	Immediate uint8
}

func (n *MoveCompareAddSubtractImmediateInstruction) String() string {
	var start string
	if n.Operation == 0 {
		start = "mov"
	} else if n.Operation == 1 {
		start = "cmp"
	} else if n.Operation == 2 {
		start = "add"
	} else {
		start = "sub"
	}
	return fmt.Sprintf("%s %s, %d", start, n.Rd, n.Immediate)
}

type ALUOperationInstruction struct {
	basicTHUMBInstruction
	Opcode ALUOpcodeTHUMB
	Rd     ARMRegister
	Rs     ARMRegister
}

func (n *ALUOperationInstruction) String() string {
	return fmt.Sprintf("%s %s, %s", n.Opcode, n.Rd, n.Rs)
}

type HighRegisterOperationInstruction struct {
	basicTHUMBInstruction
	Rd        ARMRegister
	Rs        ARMRegister
	HighFlag1 bool
	HighFlag2 bool
	Operation uint8
}

func (n *HighRegisterOperationInstruction) String() string {
	var start string
	if n.Operation == 0 {
		start = "add"
	} else if n.Operation == 1 {
		start = "cmp"
	} else if n.Operation == 2 {
		start = "mov"
	} else {
		return fmt.Sprintf("bx %s", n.Rs)
	}
	return fmt.Sprintf("%s %s, %s", start, n.Rd, n.Rs)
}

type PcRelativeLoadInstruction struct {
	basicTHUMBInstruction
	Offset uint8
	Rd     ARMRegister
}

func (n *PcRelativeLoadInstruction) String() string {
	return fmt.Sprintf("ldr %s, [pc, %d]", n.Rd, uint16(n.Offset)<<2)
}

type LoadStoreRegisterOffsetInstruction struct {
	basicTHUMBInstruction
	Rd           ARMRegister
	Rb           ARMRegister
	Ro           ARMRegister
	ByteQuantity bool
	Load         bool
}

func (n *LoadStoreRegisterOffsetInstruction) String() string {
	var start string
	if n.Load {
		start = "ldr"
	} else {
		start = "str"
	}
	if n.ByteQuantity {
		start += "b"
	}
	return fmt.Sprintf("%s %s, [%s, %s]", start, n.Rd, n.Rb, n.Ro)
}

type LoadStoreSignExtendedHalfwordInstruction struct {
	basicTHUMBInstruction
	Rd         ARMRegister
	Rb         ARMRegister
	Ro         ARMRegister
	SignExtend bool
	HBit       bool
}

func (n *LoadStoreSignExtendedHalfwordInstruction) String() string {
	var start string
	if n.SignExtend {
		if n.HBit {
			start = "ldsh"
		} else {
			start = "ldsb"
		}
	} else {
		if n.HBit {
			start = "ldrh"
		} else {
			start = "strh"
		}
	}
	return fmt.Sprintf("%s %s, [%s, %s]", start, n.Rd, n.Rb, n.Ro)
}

type LoadStoreImmediateOffsetInstruction struct {
	basicTHUMBInstruction
	Rd           ARMRegister
	Rb           ARMRegister
	Offset       uint8
	Load         bool
	ByteQuantity bool
}

func (n *LoadStoreImmediateOffsetInstruction) String() string {
	var start string
	if n.Load {
		start = "ldr"
	} else {
		start = "str"
	}
	offset := n.Offset
	if n.ByteQuantity {
		start += "b"
	} else {
		offset = offset << 2
	}
	return fmt.Sprintf("%s %s, [%s, %d]", start, n.Rd, n.Rb, offset)
}

type LoadStoreHalfwordInstruction struct {
	basicTHUMBInstruction
	Rd     ARMRegister
	Rb     ARMRegister
	Offset uint8
	Load   bool
}

func (n *LoadStoreHalfwordInstruction) String() string {
	var start string
	if n.Load {
		start = "ldrh"
	} else {
		start = "strh"
	}
	return fmt.Sprintf("%s %s, [%s, %d]", start, n.Rd, n.Rb, n.Offset<<1)
}

type SPRelativeLoadStoreInstruction struct {
	basicTHUMBInstruction
	Offset uint8
	Rd     ARMRegister
	Load   bool
}

func (n *SPRelativeLoadStoreInstruction) String() string {
	var start string
	if n.Load {
		start = "ldr"
	} else {
		start = "str"
	}
	return fmt.Sprintf("%s %s, [sp, %d]", start, n.Rd, uint16(n.Offset)<<1)
}

type LoadAddressInstruction struct {
	basicTHUMBInstruction
	Offset uint8
	Rd     ARMRegister
	LoadSP bool
}

func (n *LoadAddressInstruction) String() string {
	var source string
	if n.LoadSP {
		source = "sp"
	} else {
		source = "pc"
	}
	return fmt.Sprintf("add %s, %s, %d", n.Rd, source, uint16(n.Offset)<<2)
}

type AddToStackPointerInstruction struct {
	basicTHUMBInstruction
	Offset   uint8
	Negative bool
}

func (n *AddToStackPointerInstruction) String() string {
	offset := int(n.Offset) << 2
	if n.Negative {
		offset = -offset
	}
	return fmt.Sprintf("add sp, %d", offset)
}

type PushPopRegistersInstruction struct {
	basicTHUMBInstruction
	RegisterList  uint8
	StoreLRLoadPC bool
	Load          bool
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
	registerList := registerListStringTHUMB(n.RegisterList)
	if n.Load {
		start = "pop"
		if n.StoreLRLoadPC {
			registerList += ", pc"
		}
	} else {
		start = "push"
		if n.StoreLRLoadPC {
			registerList += ", lr"
		}
	}
	return fmt.Sprintf("%s {%s}", start, registerList)

}

type MultipleLoadStoreInstruction struct {
	basicTHUMBInstruction
	RegisterList uint8
	Rb           ARMRegister
	Load         bool
}

func (n *MultipleLoadStoreInstruction) String() string {
	var start string
	if n.Load {
		start = "ldmia"
	} else {
		start = "stmia"
	}
	registers := registerListStringTHUMB(n.RegisterList)
	return fmt.Sprintf("%s %s!, {%s}", start, n.Rb, registers)
}

type ConditionalBranchInstruction struct {
	basicTHUMBInstruction
	Offset    uint8
	Condition ARMCondition
}

func (n *ConditionalBranchInstruction) String() string {
	// Offset must be a signed type before it is converted to 16-bits
	offset := int16(int8(n.Offset)) << 1
	return fmt.Sprintf("b%s %d", n.Condition, offset)
}

type SoftwareInterruptTHUMBInstruction struct {
	basicTHUMBInstruction
	Comment uint8
}

func (n *SoftwareInterruptTHUMBInstruction) String() string {
	return fmt.Sprintf("swi %d", n.Comment)
}

type UnconditionalBranchInstruction struct {
	basicTHUMBInstruction
	Offset uint16
}

func (n *UnconditionalBranchInstruction) String() string {
	// Take care of sign extending and left-shifting by 1
	offset := int16(n.Offset<<5) >> 4
	return fmt.Sprintf("b %d", offset)
}

type LongBranchAndLinkInstruction struct {
	basicTHUMBInstruction
	Offset    uint16
	OffsetLow bool
}

func (n *LongBranchAndLinkInstruction) String() string {
	if n.OffsetLow {
		return fmt.Sprintf("bl lr + %d (long branch and link)", n.Offset<<1)
	}
	return fmt.Sprintf("add lr, pc, %d (long branch and link)",
		(int32(n.Offset)<<21)>>9)
}

func parseMoveShiftedRegisterInstruction(r uint16) (THUMBInstruction, error) {
	var toReturn MoveShiftedRegisterInstruction
	toReturn.raw = r
	toReturn.Rd = ARMRegister(uint8(r & 7))
	toReturn.Rs = ARMRegister(uint8((r >> 3) & 7))
	toReturn.Offset = uint8((r >> 6) & 0x1f)
	toReturn.Operation = uint8((r >> 11) & 3)
	return &toReturn, nil
}

func parseAddSubtractInstruction(raw uint16) (THUMBInstruction, error) {
	var toReturn AddSubtractInstruction
	toReturn.raw = raw
	toReturn.Rd = ARMRegister(uint8(raw & 7))
	toReturn.Rs = ARMRegister(uint8((raw >> 3) & 7))
	toReturn.Subtract = (raw & 0x200) != 0
	toReturn.Immediate = uint8((raw >> 6) & 7)
	toReturn.IsImmediate = (raw & 0x400) != 0
	if !toReturn.IsImmediate {
		toReturn.Rn = ARMRegister(toReturn.Immediate)
	}
	return &toReturn, nil
}

func parseMoveCompareAddSubtractImmediateInstruction(raw uint16) (
	THUMBInstruction, error) {
	var toReturn MoveCompareAddSubtractImmediateInstruction
	toReturn.raw = raw
	toReturn.Immediate = uint8(raw & 0xff)
	toReturn.Operation = uint8((raw >> 11) & 3)
	toReturn.Rd = ARMRegister(uint8((raw >> 8) & 7))
	return &toReturn, nil
}

func parseALUOperationInstruction(raw uint16) (THUMBInstruction, error) {
	var toReturn ALUOperationInstruction
	toReturn.raw = raw
	toReturn.Opcode = NewALUOpcodeTHUMB(uint8((raw >> 6) & 0xf))
	toReturn.Rd = ARMRegister(uint8(raw & 7))
	toReturn.Rs = ARMRegister(uint8((raw >> 3) & 7))
	return &toReturn, nil
}

func parseHighRegisterOperationInstruction(raw uint16) (THUMBInstruction,
	error) {
	var toReturn HighRegisterOperationInstruction
	var h bool
	var register uint8
	toReturn.raw = raw
	toReturn.Operation = uint8((raw >> 8) & 3)
	h = (raw & 0x80) != 0
	toReturn.HighFlag1 = h
	register = uint8(raw & 7)
	if h {
		register += 8
	}
	toReturn.Rd = ARMRegister(register)
	h = (raw & 0x40) != 0
	toReturn.HighFlag2 = h
	register = uint8((raw >> 3) & 7)
	if h {
		register += 8
	}
	toReturn.Rs = ARMRegister(register)
	return &toReturn, nil
}

func parsePCRelativeLoadInstruction(raw uint16) (THUMBInstruction, error) {
	var toReturn PcRelativeLoadInstruction
	toReturn.raw = raw
	toReturn.Offset = uint8(raw & 0xff)
	toReturn.Rd = ARMRegister(uint8((raw >> 8) & 7))
	return &toReturn, nil
}

func parseLoadStoreRegisterOffsetInstruction(raw uint16) (THUMBInstruction,
	error) {
	var toReturn LoadStoreRegisterOffsetInstruction
	toReturn.raw = raw
	toReturn.ByteQuantity = (raw & 0x400) != 0
	toReturn.Load = (raw & 0x800) != 0
	toReturn.Rd = ARMRegister(uint8(raw & 7))
	toReturn.Rb = ARMRegister(uint8((raw >> 3) & 7))
	toReturn.Ro = ARMRegister(uint8((raw >> 6) & 7))
	return &toReturn, nil
}

func parseLoadStoreSignExtendedHalfwordInstruction(raw uint16) (
	THUMBInstruction, error) {
	var toReturn LoadStoreSignExtendedHalfwordInstruction
	toReturn.raw = raw
	toReturn.Rd = ARMRegister(uint8(raw & 7))
	toReturn.Rb = ARMRegister(uint8((raw >> 3) & 7))
	toReturn.Ro = ARMRegister(uint8((raw >> 6) & 7))
	toReturn.SignExtend = (raw & 0x400) != 0
	toReturn.HBit = (raw & 0x800) != 0
	return &toReturn, nil
}

func parseLoadStoreImmediateOffsetInstruction(raw uint16) (THUMBInstruction,
	error) {
	var toReturn LoadStoreImmediateOffsetInstruction
	toReturn.raw = raw
	toReturn.Rd = ARMRegister(uint8(raw & 7))
	toReturn.Rb = ARMRegister(uint8((raw >> 3) & 7))
	toReturn.Offset = uint8((raw >> 6) & 0x1f)
	toReturn.Load = (raw & 0x800) != 0
	toReturn.ByteQuantity = (raw & 0x1000) != 0
	return &toReturn, nil
}

func parseLoadStoreHalfwordInstruction(raw uint16) (THUMBInstruction, error) {
	var toReturn LoadStoreHalfwordInstruction
	toReturn.raw = raw
	toReturn.Rd = ARMRegister(uint8(raw & 7))
	toReturn.Rb = ARMRegister(uint8((raw >> 3) & 7))
	toReturn.Offset = uint8((raw >> 6) & 0x1f)
	toReturn.Load = (raw & 0x800) != 0
	return &toReturn, nil
}

func parseSPRelativeLoadStoreInstruction(raw uint16) (THUMBInstruction,
	error) {
	var toReturn SPRelativeLoadStoreInstruction
	toReturn.raw = raw
	toReturn.Offset = uint8(raw & 0xff)
	toReturn.Rd = ARMRegister(uint8((raw >> 8) & 0x7))
	toReturn.Load = (raw & 0x800) != 0
	return &toReturn, nil
}

func parseLoadAddressInstruction(raw uint16) (THUMBInstruction, error) {
	var toReturn LoadAddressInstruction
	toReturn.raw = raw
	toReturn.Offset = uint8(raw & 0xff)
	toReturn.Rd = ARMRegister(uint8((raw >> 8) & 0x7))
	toReturn.LoadSP = (raw & 0x800) != 0
	return &toReturn, nil
}

func parseAddToStackPointerInstruction(raw uint16) (THUMBInstruction, error) {
	var toReturn AddToStackPointerInstruction
	toReturn.raw = raw
	toReturn.Offset = uint8(raw & 0x7f)
	toReturn.Negative = (raw & 0x80) != 0
	return &toReturn, nil
}

func parsePushPopRegistersInstruction(raw uint16) (THUMBInstruction, error) {
	var toReturn PushPopRegistersInstruction
	toReturn.raw = raw
	toReturn.RegisterList = uint8(raw & 0xff)
	toReturn.StoreLRLoadPC = (raw & 0x100) != 0
	toReturn.Load = (raw & 0x800) != 0
	return &toReturn, nil
}

func parseMultipleLoadStoreInstruction(raw uint16) (THUMBInstruction, error) {
	var toReturn MultipleLoadStoreInstruction
	toReturn.raw = raw
	toReturn.RegisterList = uint8(raw & 0xff)
	toReturn.Rb = ARMRegister(uint8((raw >> 8) & 0x7))
	toReturn.Load = (raw & 0x800) != 0
	return &toReturn, nil
}

func parseConditionalBranchInstruction(raw uint16) (THUMBInstruction, error) {
	var toReturn ConditionalBranchInstruction
	toReturn.raw = raw
	toReturn.Offset = uint8(raw & 0xff)
	toReturn.Condition = ARMCondition((raw >> 8) & 0xf)
	if toReturn.Condition == 14 {
		return &toReturn, fmt.Errorf("Illegal condition in conditional branch")
	}
	return &toReturn, nil
}

func parseSoftwareInterruptTHUMBInstruction(raw uint16) (THUMBInstruction,
	error) {
	var toReturn SoftwareInterruptTHUMBInstruction
	toReturn.raw = raw
	toReturn.Comment = uint8(raw & 0xff)
	return &toReturn, nil
}

func parseUnconditionalBranchInstruction(raw uint16) (THUMBInstruction,
	error) {
	var toReturn UnconditionalBranchInstruction
	toReturn.raw = raw
	toReturn.Offset = raw & 0x7ff
	return &toReturn, nil
}

func parseLongBranchAndLinkInstruction(raw uint16) (THUMBInstruction, error) {
	var toReturn LongBranchAndLinkInstruction
	toReturn.raw = raw
	toReturn.Offset = raw & 0x7ff
	toReturn.OffsetLow = (raw & 0x800) != 0
	return &toReturn, nil
}

func ParseTHUMBInstruction(raw uint16) (THUMBInstruction, error) {
	if (raw & 0x8000) != 0 {
		if (raw & 0x4000) != 0 {
			if (raw & 0x2000) != 0 {
				if (raw & 0x1000) != 0 {
					return parseLongBranchAndLinkInstruction(raw)
				}
				return parseUnconditionalBranchInstruction(raw)
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
