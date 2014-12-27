package arm_emulate

import (
	"testing"
)

// The following utility functions are equivalent to their counterparts in the
// arm emulation testing file, but modified to use 16-bit opcodes instead.

func setupTestTHUMBProcessor() (ARMProcessor, error) {
	p, e := setupTestProcessor()
	if e != nil {
		return nil, e
	}
	e = p.SetTHUMBMode(true)
	if e != nil {
		return nil, e
	}
	return p, nil
}

func testSingleTHUMBInstruction(raw uint16, p ARMProcessor) error {
	e := p.GetMemoryInterface().WriteMemoryHalfword(4096, raw)
	if e != nil {
		return e
	}
	e = p.SetRegister(15, 4096)
	if e != nil {
		return e
	}
	return p.RunNextInstruction()
}

func writeTHUMBInstructionsToMemory(instructions []uint16,
	p ARMProcessor) error {
	baseAddress := uint32(4096)
	for i := uint32(0); i < uint32(len(instructions)); i++ {
		e := p.GetMemoryInterface().WriteMemoryHalfword(baseAddress+(i*2),
			instructions[i])
		if e != nil {
			return e
		}
	}
	return nil
}

func TestMoveShiftedRegisterEmulation(t *testing.T) {
	p, e := setupTestTHUMBProcessor()
	if e != nil {
		t.FailNow()
	}
	p.SetRegister(0, 1337)
	p.SetRegister(1, 0)
	p.SetCarry(true)
	p.SetNegative(true)
	p.SetZero(true)
	// lsl r1, r0, 1
	e = testSingleTHUMBInstruction(0x0041, p)
	if e != nil {
		t.Logf("Failed running lsl instruction: %s\n", e)
		t.Fail()
	}
	value, _ := p.GetRegister(1)
	if value != (1337 << 1) {
		t.Logf("lsl instruction produced incorrect result: %d\n", value)
		t.Fail()
	}
	if p.Carry() || p.Negative() || p.Zero() {
		t.Logf("lsl instruction set flags incorrectly\n")
		t.Fail()
	}
	p.SetNegative(false)
	p.SetCarry(false)
	p.SetZero(true)
	p.SetRegister(0, 0x80000000)
	// asr r0, r0, 32 (32 is encoded as 0)
	e = testSingleTHUMBInstruction(0x1000, p)
	if e != nil {
		t.Logf("Failed running asr instruction: %s\n", e)
		t.Fail()
	}
	value, _ = p.GetRegister(0)
	if value != 0xffffffff {
		t.Logf("asr instruction produced incorrect result: %08x\n", value)
		t.Fail()
	}
	if !p.Carry() || !p.Negative() || p.Zero() {
		t.Logf("asr instruction failed to set correct flags.\n")
		t.Fail()
	}
	// lsr r0, r0, 31
	e = testSingleTHUMBInstruction(0x0fc0, p)
	if e != nil {
		t.Logf("Failed running lsr instruction: %s\n", e)
		t.Fail()
	}
	value, _ = p.GetRegister(0)
	if value != 1 {
		t.Logf("lsr produced a vaue of %d instead of 1.\n", value)
		t.Fail()
	}
	// asr r0, r0, 1
	e = testSingleTHUMBInstruction(0x1040, p)
	if e != nil {
		t.Logf("Failed running 2nd asr instruction: %s\n", e)
		t.Fail()
	}
	value, _ = p.GetRegister(0)
	if value != 0 {
		t.Logf("asr instruction set r0 to %d instead of 0.\n", value)
		t.Fail()
	}
	if !p.Zero() {
		t.Logf("ASR shift failed to set zero flag.\n")
		t.Fail()
	}
}

func TestAddSubtractEmulation(t *testing.T) {
	p, e := setupTestTHUMBProcessor()
	if e != nil {
		t.FailNow()
	}
	p.SetRegister(0, 2)
	p.SetRegister(1, 0xfffffffe)
	p.SetCarry(false)
	p.SetOverflow(true)
	p.SetZero(false)
	p.SetNegative(true)
	// add r0, r0, r1
	e = testSingleTHUMBInstruction(0x1840, p)
	if e != nil {
		t.Logf("Failed to run add instruction: %s\n", e)
		t.Fail()
	}
	value, _ := p.GetRegister(0)
	if value != 0 {
		t.Logf("Add produced incorrect value: %d (expected 0).\n", value)
		t.Fail()
	}
	if !p.Carry() {
		t.Logf("Add instruction should have set the carry bit.\n")
		t.Fail()
	}
	if p.Overflow() {
		t.Logf("Add instruction shouldn't have set overflow.\n")
		t.Fail()
	}
	if !p.Zero() {
		t.Logf("Add instruction should have set zero.\n")
		t.Fail()
	}
	if p.Negative() {
		t.Logf("Add instruction shouldn't have set negative.\n")
		t.Fail()
	}
	// sub r0, r0, 7
	e = testSingleTHUMBInstruction(0x1fc0, p)
	if e != nil {
		t.Logf("Failed to run sub instruction: %s\n", e)
		t.Fail()
	}
	value, _ = p.GetRegister(0)
	if int32(value) != -7 {
		t.Logf("Sub instruction got %d instead of -7.\n", int(value))
		t.Fail()
	}
	if !p.Negative() {
		t.Logf("Sub instruction should have set negative.\n")
		t.Fail()
	}
	if !p.Carry() {
		t.Logf("Sub instruction should have set carry.\n")
		t.Fail()
	}
}

func TestMoveCompareAddSubtractImmediateEmulation(t *testing.T) {
	p, e := setupTestTHUMBProcessor()
	if e != nil {
		t.FailNow()
	}
	instructions := []uint16{
		// mov r0, 125
		0x207d,
		// add r0, 225
		0x30e1,
		// sub r0, 100
		0x3864,
		// cmp r0, 250
		0x28fa}
	e = writeTHUMBInstructionsToMemory(instructions, p)
	if e != nil {
		t.FailNow()
	}
	p.SetRegister(15, 4096)
	e = runMultipleInstructions(4, p, t)
	if e != nil {
		t.Fail()
	}
	value, _ := p.GetRegister(0)
	if value != 250 {
		t.Logf("Got incorrect value in r0: %d. Expected 250.\n", value)
		t.Fail()
	}
	if !p.Zero() {
		t.Logf("Compare instruction should have set zero flag.\n")
		t.Fail()
	}
	p.SetOverflow(false)
	p.SetRegister(0, 0x80000000)
	// sub r0, 1
	e = testSingleTHUMBInstruction(0x3801, p)
	if e != nil {
		t.Logf("Failed running sub instruction: %s\n", e)
		t.Fail()
	}
	if !p.Overflow() {
		t.Logf("Sub instruction failed to set overflow.\n")
		t.Fail()
	}
	value, _ = p.GetRegister(0)
	if value != 0x7fffffff {
		t.Logf("Sub produced 0x%08x instead of 0x7fffffff\n", value)
		t.Fail()
	}
}

func TestALUOperationEmulation(t *testing.T) {
	p, e := setupTestTHUMBProcessor()
	if e != nil {
		t.FailNow()
	}
	instructions := []uint16{
		// orr r0, r1
		0x4308,
		// eor r1, r1
		0x4049,
		// ror r0, r1
		0x41c8}
	e = writeTHUMBInstructionsToMemory(instructions, p)
	if e != nil {
		t.FailNow()
	}
	p.SetRegister(15, 4096)
	p.SetRegister(0, 0)
	p.SetRegister(1, 1337)
	runMultipleInstructions(3, p, t)
	value, _ := p.GetRegister(0)
	if value != 1337 {
		t.Logf("Expected 1337 in r0, got %d\n", value)
		t.Fail()
	}
	value, _ = p.GetRegister(1)
	if value != 0 {
		t.Logf("Expected 0 in r1, got %d\n", value)
		t.Fail()
	}
	p.SetNegative(false)
	p.SetCarry(true)
	p.SetZero(true)
	p.SetRegister(0, 0x7fffffff)
	p.SetRegister(1, 0)
	// adc r0, r1
	e = testSingleTHUMBInstruction(0x4148, p)
	if e != nil {
		t.Logf("Adc instruction failed: %s\n", e)
		t.Fail()
	}
	value, _ = p.GetRegister(0)
	if value != 0x80000000 {
		t.Logf("Expected 0x80000000 in r0, got 0x%08x\n", value)
		t.Fail()
	}
	if p.Zero() {
		t.Logf("Adc shouldn't have set zero flag.\n")
		t.Fail()
	}
	if !p.Negative() {
		t.Logf("Adc should have set the negative flag.\n")
		t.Fail()
	}
	if p.Carry() {
		t.Logf("Adc shouldn't have set the carry flag.\n")
		t.Fail()
	}
}

func TestHighRegisterOperationEmulation(t *testing.T) {
	p, e := setupTestTHUMBProcessor()
	if e != nil {
		t.FailNow()
	}
	p.SetRegister(0, 1337)
	p.SetRegister(8, 0)
	p.SetRegister(9, 0)
	p.SetRegister(13, 1)
	p.SetRegister(1, 0xffffffff)
	p.SetZero(true)
	p.SetCarry(false)
	p.SetOverflow(true)
	p.SetNegative(true)
	// add r8, r0
	e = testSingleTHUMBInstruction(0x4480, p)
	if e != nil {
		t.Logf("Failed running add (high register): %s\n", e)
		t.Fail()
	}
	value, _ := p.GetRegister(8)
	if value != 1337 {
		t.Logf("Expected 1337 in r8. Got %d\n", value)
		t.Fail()
	}
	// mov r9, r13
	e = testSingleTHUMBInstruction(0x46e9, p)
	if e != nil {
		t.Logf("Failed running mov (high register): %s\n", e)
		t.Fail()
	}
	value, _ = p.GetRegister(9)
	if value != 1 {
		t.Logf("Expected 1 in r9, got %d\n", value)
		t.Fail()
	}
	if !p.Zero() || p.Carry() || !p.Overflow() || !p.Negative() {
		t.Logf("High register ops incorrectly modified flags.\n")
		t.Fail()
	}
	// cmp r9, r1
	e = testSingleTHUMBInstruction(0x4589, p)
	if e != nil {
		t.Logf("Failed running cmp (high register): %s\n", e)
		t.Fail()
	}
	if p.Zero() || !p.Carry() || p.Overflow() || p.Negative() {
		t.Logf("High-register cmp failed setting flags.\n")
		t.Fail()
	}
	p.SetRegister(14, 4090)
	// bx r14 (lr)
	e = testSingleTHUMBInstruction(0x4770, p)
	value, _ = p.GetRegister(15)
	if value != 4090 {
		t.Logf("Bx set r15 to %d instead of 4770.\n", value)
		t.Fail()
	}
	if p.THUMBMode() {
		t.Logf("Bx failed to switch to ARM mode.\n")
		t.Fail()
	}
}

func TestPCRelativeLoadEmulation(t *testing.T) {
	p, e := setupTestTHUMBProcessor()
	if e != nil {
		t.FailNow()
	}
	m := p.GetMemoryInterface()
	e = m.WriteMemoryWord(4100, 1337)
	if e != nil {
		t.FailNow()
	}
	e = m.WriteMemoryWord(4116, 100)
	if e != nil {
		t.FailNow()
	}
	p.SetRegister(0, 0)
	p.SetRegister(0, 0)
	// ldr r0, [pc, 0]
	e = testSingleTHUMBInstruction(0x4800, p)
	if e != nil {
		t.Logf("Failed running pc-relative load 1: %s\n", e)
		t.Fail()
	}
	value, _ := p.GetRegister(0)
	if value != 1337 {
		t.Logf("Read %d, not 1337, relative to PC.\n", value)
		t.Fail()
	}
	// ldr r1, [pc, 16]
	e = testSingleTHUMBInstruction(0x4904, p)
	if e != nil {
		t.Logf("Failed running pc-relative load 2: %s\n", e)
		t.Fail()
	}
	value, _ = p.GetRegister(1)
	if value != 100 {
		t.Logf("Read %d, not 100, relative to PC.\n", value)
		t.Fail()
	}
}

func TestLoadStoreRegisterOffsetEmulation(t *testing.T) {
	p, e := setupTestTHUMBProcessor()
	if e != nil {
		t.FailNow()
	}
	m := p.GetMemoryInterface()
	p.SetRegister(0, 1337)
	p.SetRegister(1, 0)
	p.SetRegister(2, 4100)
	// str r0, [r1, r2]
	e = testSingleTHUMBInstruction(0x5088, p)
	if e != nil {
		t.Logf("Failed running str: %s\n", e)
		t.Fail()
	}
	value, e := m.ReadMemoryWord(4100)
	if value != 1337 {
		t.Logf("Wrote %d instead of 1337?\n", value)
		t.Fail()
	}
	p.SetRegister(0, 0)
	p.SetRegister(1, 4000)
	p.SetRegister(2, 100)
	// ldr r0, [r1, r2]
	e = testSingleTHUMBInstruction(0x5888, p)
	if e != nil {
		t.Logf("Failed running ldr: %s\n", e)
		t.Fail()
	}
	value, _ = p.GetRegister(0)
	if value != 1337 {
		t.Logf("Expected to read 1337, got %d instead.\n", value)
		t.Fail()
	}
	p.SetRegister(0, 0x24)
	p.SetRegister(1, 4200)
	p.SetRegister(2, 13)
	// strb r0, [r1, r2]
	e = testSingleTHUMBInstruction(0x5450, p)
	if e != nil {
		t.Logf("Failed running strb: %s\n", e)
		t.Fail()
	}
	byteValue, e := m.ReadMemoryByte(4213)
	if e != nil {
		t.FailNow()
	}
	if byteValue != 0x24 {
		t.Logf("Expected to write 0x24, wrote 0x%02x instead.\n", byteValue)
		t.Fail()
	}
	p.SetRegister(0, 0)
	// ldrb r0, [r1, r2]
	e = testSingleTHUMBInstruction(0x5c50, p)
	if e != nil {
		t.Logf("Failed running ldrb: %s\n", e)
		t.Fail()
	}
	value, _ = p.GetRegister(0)
	if value != 0x24 {
		t.Logf("Expected to read 0x24, read 0x%08x instead.\n", value)
		t.Fail()
	}
}

func TestLoadStoreSignExtendedHalfwordEmulation(t *testing.T) {
	p, e := setupTestTHUMBProcessor()
	if e != nil {
		t.FailNow()
	}
	m := p.GetMemoryInterface()
	e = m.WriteMemoryByte(4111, 0x80)
	if e != nil {
		t.FailNow()
	}
	p.SetRegister(0, 0)
	p.SetRegister(1, 4111)
	// ldsb r0, [r1, r0]
	e = testSingleTHUMBInstruction(0x5608, p)
	if e != nil {
		t.Logf("Failed running ldsb instruction: %s\n", e)
		t.Fail()
	}
	value, _ := p.GetRegister(0)
	if value != 0xffffff80 {
		t.Logf("Expected to sign-extend byte to 0xffffff80, not 0x%08x\n",
			value)
		t.Fail()
	}
	p.SetRegister(0, 0xfeedfeed)
	p.SetRegister(1, 4100)
	p.SetRegister(2, 20)
	// strh r0, [r1, r2]
	e = testSingleTHUMBInstruction(0x5288, p)
	if e != nil {
		t.Logf("Faied running strh: %s\n", e)
		t.Fail()
	}
	value, e = m.ReadMemoryWord(4120)
	if e != nil {
		t.FailNow()
	}
	if value != 0xfeed {
		t.Logf("Expected to store 0xfeed, stored 0x%08x instead.\n", value)
		t.Fail()
	}
	// ldrh r0, [r1, r2]
	e = testSingleTHUMBInstruction(0x5a88, p)
	if e != nil {
		t.Logf("Failed to run ldrh: %s\n", e)
		t.Fail()
	}
	value, _ = p.GetRegister(0)
	if value != 0xfeed {
		t.Logf("Expected to load 0xfeed, got 0x%08x instead.\n", value)
		t.Fail()
	}
	// ldrsh r0, [r1, r2]
	e = testSingleTHUMBInstruction(0x5e88, p)
	if e != nil {
		t.Logf("Failed to run ldrsh: %s\n", e)
		t.Fail()
	}
	value, _ = p.GetRegister(0)
	if value != 0xfffffeed {
		t.Logf("Expected to load 0xfffffeed, got 0x%08x instead.\n", value)
		t.Fail()
	}
}

func TestLoadStoreImmediateOffsetEmulation(t *testing.T) {
	p, e := setupTestTHUMBProcessor()
	if e != nil {
		t.FailNow()
	}
	m := p.GetMemoryInterface()
	instructions := []uint16{
		// ldrb r0 [r1, 0]
		0x7808,
		// ldr r2, [r1, 4]
		0x684a,
		// str r0, [r1, 8]
		0x6088,
		// strb r2, [r1, 12]
		0x730a}
	p.SetRegister(1, 4200)
	p.SetRegister(15, 4096)
	e = m.WriteMemoryWord(4200, 0x1337)
	if e != nil {
		t.FailNow()
	}
	e = m.WriteMemoryWord(4204, 0xbeef)
	if e != nil {
		t.FailNow()
	}
	e = writeTHUMBInstructionsToMemory(instructions, p)
	if e != nil {
		t.FailNow()
	}
	e = runMultipleInstructions(4, p, t)
	if e != nil {
		t.Logf("Failed running instructions: %s\n", e)
		t.Fail()
	}
	value, e := m.ReadMemoryWord(4208)
	if e != nil {
		t.FailNow()
	}
	if value != 0x37 {
		t.Logf("Expected to read 0x37 at 4208, got 0x%08x instead.\n", value)
		t.Fail()
	}
	value, e = m.ReadMemoryWord(4212)
	if value != 0xef {
		t.Logf("Expected to read 0xef at 4212, got 0x%08x instead.\n", value)
		t.Fail()
	}
}

func TestLoadStoreHalfwordEmulation(t *testing.T) {
	p, e := setupTestTHUMBProcessor()
	if e != nil {
		t.FailNow()
	}
	m := p.GetMemoryInterface()
	e = m.WriteMemoryWord(4204, 0xfeedbeef)
	if e != nil {
		t.FailNow()
	}
	p.SetRegister(0, 0)
	p.SetRegister(1, 4200)
	// ldrh r0, [r1, 4]
	e = testSingleTHUMBInstruction(0x8888, p)
	if e != nil {
		t.Logf("Failed running ldrh instruction: %s\n", e)
		t.Fail()
	}
	value, _ := p.GetRegister(0)
	if value != 0xbeef {
		t.Logf("Expected to read 0xbeef, got 0x%08x instead.\n", value)
		t.Fail()
	}
	p.SetRegister(0, 0xdecafbad)
	// strh r0, [r1, 8]
	e = testSingleTHUMBInstruction(0x8108, p)
	if e != nil {
		t.Logf("Failed running strh: %s\n", e)
		t.Fail()
	}
	value, e = m.ReadMemoryWord(4208)
	if e != nil {
		t.FailNow()
	}
	if value != 0xfbad {
		t.Logf("Expected to find 0xfbad in memory. Found 0x%08x.\n", value)
		t.Fail()
	}
}

func TestSPRelativeLoadStoreEmulation(t *testing.T) {
	p, e := setupTestTHUMBProcessor()
	if e != nil {
		t.FailNow()
	}
	m := p.GetMemoryInterface()
	e = m.WriteMemoryWord(4204, 1337)
	if e != nil {
		t.FailNow()
	}
	p.SetRegister(13, 4200)
	// ldr r0, [sp, 4]
	e = testSingleTHUMBInstruction(0x9801, p)
	if e != nil {
		t.Logf("Failed running sp-relative ldr: %s\n", e)
		t.Fail()
	}
	value, _ := p.GetRegister(0)
	if value != 1337 {
		t.Logf("Expected to read 1337, got %d.\n", value)
		t.Fail()
	}
	p.SetRegister(0, 100)
	// str r0, [sp]
	e = testSingleTHUMBInstruction(0x9000, p)
	if e != nil {
		t.Logf("Failed running sp-relative str: %s\n", e)
		t.Fail()
	}
	value, e = m.ReadMemoryWord(4200)
	if e != nil {
		t.FailNow()
	}
	if value != 100 {
		t.Logf("Expected to store 100, stored %d instead.\n", value)
		t.Fail()
	}
}

func TestLoadAddressEmulation(t *testing.T) {
	p, e := setupTestTHUMBProcessor()
	if e != nil {
		t.FailNow()
	}
	// add r0, pc, 36
	e = testSingleTHUMBInstruction(0xa009, p)
	if e != nil {
		t.Logf("Failed getting pc address: %s\n", e)
		t.Fail()
	}
	value, _ := p.GetRegister(0)
	if value != 4136 {
		t.Logf("Expected to get address 4136 from PC, got %d.\n", value)
		t.Fail()
	}
	p.SetRegister(13, 0xf0000)
	// add r0, sp, 16
	e = testSingleTHUMBInstruction(0xa804, p)
	if e != nil {
		t.Logf("Failed getting sp address: %s\n", e)
		t.Fail()
	}
	value, _ = p.GetRegister(0)
	if value != 0xf0010 {
		t.Logf("Expected 0xf0010 from sp. Got 0x%08x.\n", value)
		t.Fail()
	}
}

func TestAddToStackPointerEmulation(t *testing.T) {
	p, e := setupTestTHUMBProcessor()
	if e != nil {
		t.FailNow()
	}
	p.SetRegister(13, 128)
	// add sp, 100
	e = testSingleTHUMBInstruction(0xb019, p)
	if e != nil {
		t.Logf("Failed adding to sp: %s\n", e)
		t.Fail()
	}
	value, _ := p.GetRegister(13)
	if value != 228 {
		t.Logf("Expected sp to be 228, was %d.\n", value)
		t.Fail()
	}
	// add sp, -128
	e = testSingleTHUMBInstruction(0xb0a0, p)
	if e != nil {
		t.Logf("Failed subtracting from sp: %s\n", e)
		t.Fail()
	}
	value, _ = p.GetRegister(13)
	if value != 100 {
		t.Logf("Expected sp to be 100, was %d.\n", value)
		t.Fail()
	}
}

func TestPushPopRegistersEmulation(t *testing.T) {
	p, e := setupTestTHUMBProcessor()
	if e != nil {
		t.FailNow()
	}
	p.SetRegister(0, 1337)
	p.SetRegister(1, 2336)
	p.SetRegister(2, 0)
	p.SetRegister(3, 0)
	p.SetRegister(13, 4200)
	p.SetRegister(14, 0xf000)
	p.SetRegister(15, 4096)
	instructions := []uint16{
		// push {r0, r1, lr}
		0xb503,
		// pop {r2, r3, pc}
		0xbd0c}
	e = writeTHUMBInstructionsToMemory(instructions, p)
	if e != nil {
		t.FailNow()
	}
	e = runMultipleInstructions(2, p, t)
	if e != nil {
		t.Logf("Failed running push/pop instructions.\n")
		t.Fail()
	}
	value, _ := p.GetRegister(2)
	if value != 1337 {
		t.Logf("Expected 1337 in r2, got %d.\n", value)
		t.Fail()
	}
	value, _ = p.GetRegister(3)
	if value != 2336 {
		t.Logf("Expected 2336 in r3, got %d.\n", value)
		t.Fail()
	}
}

func TestMultipleLoadStoreEmulation(t *testing.T) {
	p, e := setupTestTHUMBProcessor()
	if e != nil {
		t.FailNow()
	}
	p.SetRegister(0, 4200)
	m := p.GetMemoryInterface()
	e = m.WriteMemoryWord(4200, 1337)
	if e != nil {
		t.FailNow()
	}
	e = m.WriteMemoryWord(4204, 2336)
	if e != nil {
		t.FailNow()
	}
	// ldmia r0!, {r1, r7}
	e = testSingleTHUMBInstruction(0xc882, p)
	if e != nil {
		t.Logf("Failed running ldmia: %s\n", e)
		t.Fail()
	}
	value, _ := p.GetRegister(1)
	if value != 1337 {
		t.Logf("ldmia loaded %d instead of 1337.\n", value)
		t.Fail()
	}
	value, _ = p.GetRegister(7)
	if value != 2336 {
		t.Logf("ldmia loaded %d instead of 2336.\n", value)
	}
	p.SetRegister(0, 4300)
	// stmia r0!, {r1, r7}
	e = testSingleTHUMBInstruction(0xc082, p)
	value, e = m.ReadMemoryWord(4300)
	if e != nil {
		t.FailNow()
	}
	if value != 1337 {
		t.Logf("stmia stored %d, not 1337.\n", value)
		t.Fail()
	}
	value, e = m.ReadMemoryWord(4304)
	if value != 2336 {
		t.Logf("stmia stored %d, not 2336.\n", value)
		t.Fail()
	}
}

func TestConditionalBranchEmulation(t *testing.T) {
	p, e := setupTestTHUMBProcessor()
	instructions := []uint16{
		// bne next
		0xd101,
		// mov r0 16
		0x2010,
		// mov r1, 32
		0x2120,
		// next: cmp r0, r0
		0x4280,
		// bne next
		0xd1fd,
		// mov r0, 97
		0x2061}
	e = writeTHUMBInstructionsToMemory(instructions, p)
	if e != nil {
		t.Logf("Failed to write instructions to memory.\n")
		t.Fail()
	}
	p.SetRegister(15, 4096)
	e = runMultipleInstructions(4, p, t)
	if e != nil {
		t.Logf("Failed running conditional branch program.\n")
		t.Fail()
	}
	value, _ := p.GetRegister(0)
	if value != 97 {
		t.Logf("Expected to get 97 in r0, got %d.\n", value)
		t.Fail()
	}
}

func TestSoftwareInterruptTHUMBEmulation(t *testing.T) {
	p, e := setupTestTHUMBProcessor()
	e = p.SetMode(0x10)
	if e != nil {
		t.FailNow()
	}
	// swi 101
	e = testSingleTHUMBInstruction(0xdf65, p)
	if e != nil {
		t.Logf("Failed running software interrupt: %s\n", e)
		t.Fail()
	}
	value, _ := p.GetRegister(15)
	if value != 0x8 {
		t.Logf("SWI set pc to 0x%08x, not 0x8.\n", value)
		t.Fail()
	}
	if p.THUMBMode() {
		t.Logf("SWI from THUMB mode didn't switch to ARM.\n")
		t.Fail()
	}
	newMode := p.GetMode()
	if newMode != 0x13 {
		t.Logf("Processor in mode 0x%02x after swi, not 0x13.\n", newMode)
		t.Fail()
	}
	value, _ = p.GetRegister(14)
	if value != 4098 {
		t.Logf("Swi incorrectly set lr to %d, not 4098.\n", value)
		t.Fail()
	}
}

func TestUnconditionalBranchEmulation(t *testing.T) {
	p, e := setupTestTHUMBProcessor()
	if e != nil {
		t.FailNow()
	}
	// branch back over 4 instructions
	e = testSingleTHUMBInstruction(0xe7fb, p)
	if e != nil {
		t.Logf("Unconditional branch (backward) failed: %s\n", e)
		t.Fail()
	}
	value, _ := p.GetRegister(15)
	if value != 4090 {
		t.Logf("Expected pc to be %d, was %d.\n", 4100-28, value)
		t.Fail()
	}
	// branch forward over 3 instructions
	e = testSingleTHUMBInstruction(0xe002, p)
	if e != nil {
		t.Logf("Unconditional branch (forward) failed: %s\n", e)
		t.Fail()
	}
	value, _ = p.GetRegister(15)
	if value != 4104 {
		t.Logf("Expected pc to be %d, was %d.\n", 4100+1338, value)
		t.Fail()
	}
}

func TestLongBranchAndLinkInstruction(t *testing.T) {
	p, e := setupTestTHUMBProcessor()
	if e != nil {
		t.FailNow()
	}
	instructions := []uint16{
		// bl -6
		0xf7ff,
		0xfffc}
	e = writeTHUMBInstructionsToMemory(instructions, p)
	if e != nil {
		t.FailNow()
	}
	p.SetRegister(15, 4096)
	e = runMultipleInstructions(2, p, t)
	if e != nil {
		t.Fail()
	}
	value, _ := p.GetRegister(15)
	if value != 4092 {
		t.Logf("Expected to jump to 4092, got %d.\n", value)
		t.Fail()
	}
	value, _ = p.GetRegister(14)
	if value != 4101 {
		t.Logf("Expected 4101 in LR, got %d.\n", value)
		t.Fail()
	}
}
