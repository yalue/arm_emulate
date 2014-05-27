package arm_emulate

import (
	"testing"
)

// Returns a new ARMProcessor with 1 page of memory mapped at offset 4096 and
// initialized to zeros.
func setupTestProcessor() (ARMProcessor, error) {
	processor := NewARMProcessor()
	blankMemory := make([]byte, 4096)
	e := processor.GetMemoryInterface().SetMemoryRegion(4096, blankMemory)
	if e != nil {
		return nil, e
	}
	return processor, nil
}

// Attempts to write the given raw instruction to offset 4096 and run it.
// Returns any error generated in the process.
func testSingleInstruction(raw uint32, p ARMProcessor) error {
	e := p.GetMemoryInterface().WriteMemoryWord(4096, raw)
	if e != nil {
		return e
	}
	e = p.SetRegisterNumber(15, 4096)
	if e != nil {
		return e
	}
	return p.RunNextInstruction()
}

// Attempts to map the given instructions to offset 4096. Returns an error if
// one occurs.
func writeInstructionsToMemory(instructions []uint32, p ARMProcessor) error {
	baseAddress := uint32(4096)
	for i := uint32(0); i < uint32(len(instructions)); i++ {
		e := p.GetMemoryInterface().WriteMemoryWord(baseAddress+(i*4),
			instructions[i])
		if e != nil {
			return e
		}
	}
	return nil
}

// Runs the given number of instructions on the processor, producing a trace
// and returning an error if one occurred.
func runMultipleInstructions(count int, p ARMProcessor, t *testing.T) error {
	for i := 0; i < count; i++ {
		t.Logf("Next instruction: %s\n", p.PendingInstructionString())
		e := p.RunNextInstruction()
		if e != nil {
			t.Logf("Failed running instruction: %s\n", e)
			return e
		}
	}
	return nil
}

func TestDataProcessingEmulation(t *testing.T) {
	// mov r0, 256
	p, e := setupTestProcessor()
	if e != nil {
		t.FailNow()
	}
	instructions := []uint32{
		// mov r0, 256
		0xe3a00c01,
		// lsl r0, r0, 2
		0xe1a00100,
		// mov r1, 1
		0xe3a01001,
		// mov r2, 156
		0xe3a0209c,
		// add r0, r0, r2 lsl r1
		0xe0800112,
		// add r0, r0, 4
		0xe2800004,
		// sub r0, r0, 3
		0xe2400003,
		// mov r0, 1
		0xe3a00001,
		// lsl r1, r0, 31
		0xe1a01f80,
		// orrs r0, r0, r1
		0xe1900001,
		// mov r1, r0
		0xe1a01000,
		// cmp r1, r0
		0xe1510000}
	e = writeInstructionsToMemory(instructions, p)
	if e != nil {
		t.FailNow()
	}
	e = p.SetRegisterNumber(15, 4096)
	if e != nil {
		t.FailNow()
	}
	// Run from the beginning through the "sub" instruction
	e = runMultipleInstructions(7, p, t)
	if e != nil {
		t.Fail()
	}
	value, _ := p.GetRegisterNumber(0)
	if value != 1337 {
		t.Fail()
	}
	// Now test the logical instructions
	p.SetNegative(false)
	for i := 0; i < 2; i++ {
		e = p.RunNextInstruction()
		if e != nil {
			t.Logf("Failed running instruction: %s\n", e)
			t.Fail()
		}
	}
	if p.Negative() {
		t.Logf("Expected the N flag to be clear.\n")
		t.Fail()
	}
	e = p.RunNextInstruction()
	if e != nil {
		t.Logf("Failed running instruction: %s\n", e)
		t.Fail()
	}
	value, _ = p.GetRegisterNumber(0)
	if value != 0x80000001 {
		t.Logf("Expected r0 to be 0x80000001, got 0x%08x\n", value)
		t.Fail()
	}
	if !p.Negative() {
		t.Logf("Expected the N flag to be set.\n")
		t.Fail()
	}
	e = runMultipleInstructions(2, p, t)
	if e != nil {
		t.Fail()
	}
	value, _ = p.GetRegisterNumber(1)
	if value != 0x80000001 {
		t.Logf("Incorrect value in r1. Expected 0x80000001.")
		t.Fail()
	}
	if !p.Zero() {
		t.Logf("cmp instruction didn't set zero flag as expected.\n")
		t.Fail()
	}
	// Test using r15: mov r0, pc
	e = testSingleInstruction(0xe1a0000f, p)
	if e != nil {
		t.Logf("Failed running mov r0, r15: %s\n", e)
		t.Fail()
	}
	value, _ = p.GetRegisterNumber(0)
	if value != 4104 {
		t.Logf("Improper value moved from r15: %d (expected 4104)\n", value)
		t.Fail()
	}
	// Check carry and overflow flag stuff
	p.SetCarry(false)
	p.SetRegisterNumber(0, 0xffffffff)
	p.SetRegisterNumber(1, 1)
	e = testSingleInstruction(0xe0900001, p)
	if e != nil {
		t.Logf("Failed running adds instruction: %s\n", e)
		t.Fail()
	}
	if !p.Carry() {
		t.Logf("Adds didn't set carry correctly.\n")
		t.Fail()
	}
	p.SetOverflow(false)
	p.SetRegisterNumber(0, 0x7fffffff)
	p.SetRegisterNumber(1, 1)
	e = testSingleInstruction(0xe0900001, p)
	if e != nil {
		t.Logf("Failed running adds instruction (2nd): %s\n", e)
		t.Fail()
	}
	if !p.Overflow() {
		t.Logf("Adds didn't set overflow correctly.\n")
		t.Fail()
	}
}

func TestPSRTransferEmulation(t *testing.T) {
	p, e := setupTestProcessor()
	if e != nil {
		t.FailNow()
	}
	e = p.SetMode(0x13)
	if e != nil {
		t.Logf("Failed setting supervisor mode: %s\n", e)
		t.FailNow()
	}
	instructions := []uint32{
		// mrs r0, spsr
		0xe14f0000,
		// msr cpsr_flg, 0xf0000000
		0xe328f20f,
		// mrs r0, cpsr
		0xe10f0000,
		// bic r0, r0, 0x1f
		0xe3c0001f,
		// orr r0, r0, 0x10
		0xe3800010,
		// msr cpsr, r0
		0xe129f000,
		// mrs r0, cpsr
		0xe10f0000,
		// bic r0, r0, 0xf0000000
		0xe3c0020f,
		// msr cpsr, r0
		0xe129f000}
	e = writeInstructionsToMemory(instructions, p)
	if e != nil {
		t.FailNow()
	}
	p.SetRegisterNumber(15, 4096)
	e = runMultipleInstructions(2, p, t)
	if e != nil {
		t.Fail()
	}
	if !p.Zero() || !p.Carry() || !p.Overflow() || !p.Negative() {
		t.Logf("Writing to cpsr didn't change flags properly.\n")
		t.Fail()
	}
	e = runMultipleInstructions(4, p, t)
	if e != nil {
		t.Fail()
	}
	if p.GetMode() != 0x10 {
		t.Logf("Writing to cpsr failed to change mode properly.\n")
		t.Fail()
	}
}

// This also servers as the test for SetRegisterNumber and GetRegisterNumber.
// Checking for errors from those functions everywhere is excessive. This test
// should probably include an example of each instruction, too...
func TestConditionalExecutionEmulation(t *testing.T) {
	p, e := setupTestProcessor()
	if e != nil {
		t.FailNow()
	}
	e = p.SetRegisterNumber(0, 1337)
	p.SetZero(true)
	// movne r0, 0
	e = testSingleInstruction(0x13a00000, p)
	if e != nil {
		t.Fail()
	}
	value, e := p.GetRegisterNumber(0)
	if e != nil {
		t.Fail()
	}
	if (e != nil) || (value != 1337) {
		t.Logf("Conditional instruction modified register when it shouldn't\n")
		t.Fail()
	}
	// moveq r0, 0
	e = testSingleInstruction(0x03a00000, p)
	if e != nil {
		t.Fail()
	}
	value, e = p.GetRegisterNumber(0)
	if (e != nil) || (value != 0) {
		t.Logf("Conditional instruction failed to modify register.\n")
		t.Fail()
	}
}

func TestMultiplyEmulation(t *testing.T) {
	p, e := setupTestProcessor()
	if e != nil {
		t.FailNow()
	}
	p.SetZero(true)
	p.SetRegisterNumber(0, 668)
	p.SetRegisterNumber(1, 2)
	p.SetRegisterNumber(2, 1)
	p.SetRegisterNumber(3, 0)
	// mul r3, r0, r1
	e = testSingleInstruction(0xe0030190, p)
	if e != nil {
		t.Logf("Error running mul: %s\n", e)
		t.Fail()
	}
	value, _ := p.GetRegisterNumber(3)
	if value != 1336 {
		t.Logf("Wrong multiplication result. Got %d instead of 1336.\n", value)
		t.Fail()
	}
	if !p.Zero() {
		t.Logf("Multiply instruction modified flags when it shouldn't.\n")
		t.Fail()
	}
	// mlas r3, r0, r1, r2
	e = testSingleInstruction(0xe0332190, p)
	if e != nil {
		t.Logf("Error running mlas: %s\n", e)
		t.Fail()
	}
	value, _ = p.GetRegisterNumber(3)
	if value != 1337 {
		t.Logf("Wrong result for mlas: %d. Expected 1337.\n", value)
		t.Fail()
	}
	if p.Zero() {
		t.Logf("mlas incorrectly modified flags.\n")
		t.Fail()
	}
	p.SetRegisterNumber(0, 0xffffffff)
	p.SetRegisterNumber(1, 2)
	p.SetRegisterNumber(2, 0)
	p.SetRegisterNumber(3, 0)
	// umull r3, r2, r0, r1
	e = testSingleInstruction(0xe0823190, p)
	if e != nil {
		t.Logf("Error running umull: %s\n", e)
		t.Fail()
	}
	value, _ = p.GetRegisterNumber(2)
	fullValue := uint64(value) << 32
	value, _ = p.GetRegisterNumber(3)
	fullValue |= uint64(value)
	if fullValue != (0xffffffff * 2) {
		t.Logf("Incorrect umull result: %016x\n", fullValue)
		t.Fail()
	}
	// "Accumulate" -1 in addition to the multiplication result
	p.SetRegisterNumber(2, 0xffffffff)
	p.SetRegisterNumber(3, 0xffffffff)
	// smlal r3, r2, r0, r1
	e = testSingleInstruction(0xe0e23190, p)
	if e != nil {
		t.Logf("Error running smlal: %s\n", e)
		t.Fail()
	}
	value, _ = p.GetRegisterNumber(2)
	signedFullValue := int64(value) << 32
	value, _ = p.GetRegisterNumber(3)
	// First convert to uint64 to avoid sign extension here
	signedFullValue |= int64(uint64(value))
	if signedFullValue != ((-1 * 2) - 1) {
		t.Logf("Incorrect smlal result: %016x\n", signedFullValue)
		t.Fail()
	}
}

func TestSingleDataSwapEmulation(t *testing.T) {
	p, e := setupTestProcessor()
	if e != nil {
		t.FailNow()
	}
	memory := p.GetMemoryInterface()
	e = memory.WriteMemoryWord(5000, 0x13371337)
	if e != nil {
		t.Fail()
	}
	p.SetRegisterNumber(2, 5000)
	p.SetRegisterNumber(0, 0xdeadbeef)
	// swp r0, r0, [r2]
	e = testSingleInstruction(0xe1020090, p)
	if e != nil {
		t.Logf("Failed running swp instruction: %s\n", e)
		t.Fail()
	}
	value, _ := p.GetRegisterNumber(0)
	t.Logf("Got value 0x%08x in r0.\n", value)
	if value != 0x13371337 {
		t.Logf("Failed swapping word from memory.\n")
		t.Fail()
	}
	value, e = memory.ReadMemoryWord(5000)
	t.Logf("Got value 0x%08x in memory.\n", value)
	if (e != nil) || (value != 0xdeadbeef) {
		t.Logf("Failed swapping word to memory.\n")
		t.Fail()
	}
	e = memory.WriteMemoryByte(5000, 0xba)
	if e != nil {
		t.Fail()
	}
	p.SetRegisterNumber(0, 0)
	p.SetRegisterNumber(1, 0x13)
	// swpb r0, r1, [r2]
	e = testSingleInstruction(0xe1420091, p)
	if e != nil {
		t.Logf("Failed running swpb instruction: %s\n", e)
		t.Fail()
	}
	value, _ = p.GetRegisterNumber(1)
	if value != 0x13 {
		t.Logf("swpb wrote to a register it shouldn't have.\n")
		t.Fail()
	}
	value, _ = p.GetRegisterNumber(0)
	if value != 0xba {
		t.Logf("swpb didn't write to the register is should have.\n")
		t.Fail()
	}
	memoryByte, e := memory.ReadMemoryByte(5000)
	if (e != nil) || (memoryByte != 0x13) {
		t.Logf("swpb didn't write the proper value to memory.\n")
		t.Fail()
	}
}

func TestBranchExchangeEmulation(t *testing.T) {
	p, e := setupTestProcessor()
	if e != nil {
		t.FailNow()
	}
	instructions := []uint32{
		// bx r0
		0xe12fff10,
		// mov r1, 0
		0xe3a01000,
		// mov r1, 137
		0xe3a01089}
	e = writeInstructionsToMemory(instructions, p)
	if e != nil {
		t.FailNow()
	}
	// Branch over the mov r1, 0 instruction
	p.SetRegisterNumber(15, 4096)
	p.SetRegisterNumber(0, 4104)
	p.SetRegisterNumber(1, 0xffffffff)
	e = runMultipleInstructions(2, p, t)
	if e != nil {
		t.Fail()
	}
	value, _ := p.GetRegisterNumber(1)
	if value != 137 {
		t.Logf("Error with bx instruction. r1 contains %d, not 137.\n", value)
		t.Fail()
	}
	p.SetRegisterNumber(0, 4096|1)
	if p.THUMBMode() {
		t.Logf("Processor shouldn't start in THUMB mode!\n")
		t.Fail()
	}
	e = testSingleInstruction(0xe12fff10, p)
	if e != nil {
		t.Logf("Error running bx instruction: %s\n", e)
		t.Fail()
	}
	if !p.THUMBMode() {
		t.Logf("Failed switching to THUMB mode using bx.\n")
		t.Fail()
	}
}

func TestHalfwordDataTransferEmulation(t *testing.T) {
	p, e := setupTestProcessor()
	if e != nil {
		t.FailNow()
	}
	// This won't be used as instructions, but as an array written to byte
	// 4096. Remember endianness: in big endian, reading from offset 4096 + 8
	// yields 0x4444 in this case, not 0x3333.
	data := []uint32{0, 0xddddffff, 0x11112222, 0x33334444}
	e = writeInstructionsToMemory(data, p)
	if e != nil {
		t.FailNow()
	}
	// ldrh r0, [pc, 4]
	e = testSingleInstruction(0xe1df00b4, p)
	if e != nil {
		t.Logf("Failed running ldrh: %s\n", e)
		t.Fail()
	}
	value, _ := p.GetRegisterNumber(0)
	if value != 0x4444 {
		t.Logf("Got wrong value from ldrh: 0x%08x\n", value)
		t.Fail()
	}
	p.SetRegisterNumber(1, 4104)
	// ldrsb r0, [r1, -4]
	e = testSingleInstruction(0xe15100d4, p)
	if e != nil {
		t.Logf("Failed running ldrsb: %s\n", e)
		t.Fail()
	}
	value, _ = p.GetRegisterNumber(0)
	if value != 0xffffffff {
		t.Logf("Got wrong value from ldrsb: 0x%08x\n", value)
		t.Fail()
	}
	p.SetRegisterNumber(0, 0x1337)
	p.SetRegisterNumber(1, 4100)
	// strh r0, [r1, 8]
	e = testSingleInstruction(0xe1c100b8, p)
	if e != nil {
		t.Logf("Failed running strh: %s\n", e)
		t.Fail()
	}
	storedValue, e := p.GetMemoryInterface().ReadMemoryHalfword(4108)
	if e != nil {
		t.Logf("Failed reading memory?\n")
		t.Fail()
	}
	if storedValue != 0x1337 {
		t.Logf("Stored 0x%04x instead of 0x1337 to memory.\n", storedValue)
		t.Fail()
	}
	p.SetRegisterNumber(1, 4100)
	p.SetRegisterNumber(2, 6)
	// ldrsh r0, [r1, r2]!
	e = testSingleInstruction(0xe1b100f2, p)
	if e != nil {
		t.Logf("Failed running ldrsh with preindexing and writeback: %s\n", e)
		t.Fail()
	}
	value, _ = p.GetRegisterNumber(0)
	if value != 0x1111 {
		t.Logf("Read 0x%04x instead of 0x1111\n", value)
		t.Fail()
	}
	value, _ = p.GetRegisterNumber(1)
	if value != 4106 {
		t.Logf("Failed to write preindexed value back.\n")
		t.Fail()
	}
	p.SetRegisterNumber(0, 0)
	// ldrsh r0, [r1], r2
	e = testSingleInstruction(0xe09100f2, p)
	if e != nil {
		t.Logf("Failed running ldrsh with postindexing: %s\n", e)
		t.Fail()
	}
	value, _ = p.GetRegisterNumber(0)
	if value != 0x1111 {
		t.Logf("Read 0x%04x instead of 0x1111 (postindex)\n", value)
		t.Fail()
	}
	value, _ = p.GetRegisterNumber(1)
	if value != 4112 {
		t.Logf("Failed to write back postindexed value.\n")
		t.Fail()
	}
}

func TestSingleDataTransferEmulation(t *testing.T) {
	p, e := setupTestProcessor()
	if e != nil {
		t.FailNow()
	}
	data := []uint32{0, 0xddddffff, 0x11112222, 0x33334444}
	e = writeInstructionsToMemory(data, p)
	if e != nil {
		t.FailNow()
	}
	p.SetRegisterNumber(0, 0)
	// ldr r0, [pc, -4]
	e = testSingleInstruction(0xe51f0004, p)
	if e != nil {
		t.Logf("Failed running ldr instruction: %s\n", e)
		t.Fail()
	}
	value, _ := p.GetRegisterNumber(0)
	if value != 0xddddffff {
		t.Logf("ldr failed. Read 0x%08x, not 0xddddffff.\n", value)
		t.Fail()
	}
	p.SetRegisterNumber(0, 0x13371337)
	p.SetRegisterNumber(1, 4100)
	// str r0, [r1, 4]!
	e = testSingleInstruction(0xe5a10004, p)
	if e != nil {
		t.Logf("Failed running str: %s\n", e)
		t.Fail()
	}
	value, e = p.GetMemoryInterface().ReadMemoryWord(4104)
	if value != 0x13371337 {
		t.Logf("str stored 0x%08x, not 0x13371337.\n", value)
		t.Fail()
	}
	value, _ = p.GetRegisterNumber(1)
	if value != 4104 {
		t.Logf("Writeback in str failed. r1 contains %d.\n", value)
		t.Fail()
	}
	p.SetRegisterNumber(0, 0x13)
	p.SetRegisterNumber(1, 4100)
	// strb r0, r1
	e = testSingleInstruction(0xe5c10000, p)
	if e != nil {
		t.Logf("Failed running strb instruction: %s\n", e)
		t.Fail()
	}
	byteValue, e := p.GetMemoryInterface().ReadMemoryByte(4100)
	if e != nil {
		t.Logf("Failed reading back byte? (%s)\n", e)
		t.Fail()
	}
	if byteValue != 0x13 {
		t.Logf("strb stored 0x%02x, not 0x13.\n", byteValue)
		t.Fail()
	}
	p.SetRegisterNumber(1, 4100)
	// ldrb r0, [r1], 3
	e = testSingleInstruction(0xe4d10003, p)
	if e != nil {
		t.Logf("Failed running strb: %s\n", e)
		t.Fail()
	}
	value, _ = p.GetRegisterNumber(0)
	if value != 0x13 {
		t.Logf("ldrb loaded 0x%08x, not 0x13\n", value)
		t.Fail()
	}
	value, _ = p.GetRegisterNumber(1)
	if value != 4103 {
		t.Logf("ldrb with postindex didn't write back properly.\n")
		t.Fail()
	}
}

func TestBlockDataTransferEmulation(t *testing.T) {
	p, e := setupTestProcessor()
	if e != nil {
		t.FailNow()
	}
	p.SetRegisterNumber(13, 4112)
	p.SetRegisterNumber(0, 1337)
	p.SetRegisterNumber(1, 2337)
	p.SetRegisterNumber(2, 0)
	p.SetRegisterNumber(3, 0)
	// stmdb sp!, {r0, r1} (push {r0, r1})
	e = testSingleInstruction(0xe92d0003, p)
	if e != nil {
		t.Logf("Failed to execute stmdb instruction: %s\n", e)
		t.Fail()
	}
	value, _ := p.GetRegisterNumber(13)
	if value != 4104 {
		t.Logf("After push, sp was at %d, not 4104\n", value)
		t.Fail()
	}
	// ldmia sp!, {r2, r3} (pop {r2, r3})
	e = testSingleInstruction(0xe8bd000c, p)
	if e != nil {
		t.Logf("Failed to execute stmia instruction: %s\n", e)
		t.Fail()
	}
	value, _ = p.GetRegisterNumber(13)
	if value != 4112 {
		t.Logf("After pop, sp was at %d, not 4112\n", value)
		t.Fail()
	}
	value, _ = p.GetRegisterNumber(2)
	if value != 1337 {
		t.Logf("Popped %d into r2, not 1337.\n", value)
		t.Fail()
	}
	value, _ = p.GetRegisterNumber(3)
	if value != 2337 {
		t.Logf("Popped %d into r3, not 2337.\n", value)
		t.Fail()
	}
	p.SetRegisterNumber(0, 4112)
	memory := p.GetMemoryInterface()
	memory.WriteMemoryWord(4112, 42)
	memory.WriteMemoryWord(4108, 84)
	// ldmda r0, {r1, r2}
	e = testSingleInstruction(0xe8100006, p)
	if e != nil {
		t.Logf("Failed running ldmda instruction: %s\n", e)
		t.Fail()
	}
	value, _ = p.GetRegisterNumber(0)
	if value != 4112 {
		t.Logf("Modified base register with writeback disabled.\n")
		t.Fail()
	}
	value, _ = p.GetRegisterNumber(2)
	if value != 42 {
		t.Logf("Got %d instead of 42 in r2.\n", value)
		t.Fail()
	}
	value, _ = p.GetRegisterNumber(1)
	if value != 84 {
		t.Logf("Got %d instead of 84 in r1.\n", value)
		t.Fail()
	}
	p.SetRegisterNumber(8, 0)
	e = p.SetMode(fiqMode)
	if e != nil {
		t.Logf("Failed to switch processor to FIQ mode.\n")
		t.FailNow()
	}
	p.SetRegisterNumber(0, 4112)
	p.SetRegisterNumber(8, 0)
	// ldmda r0, {r8}^
	e = testSingleInstruction(0xe8500100, p)
	if e != nil {
		t.Logf("Failed running ldmda to user-bank registers: %s\n", e)
		t.Fail()
	}
	value, _ = p.GetRegisterNumber(8)
	if value != 0 {
		t.Logf("Modified r8 in the fiq bank. (value: %d)\n", value)
		t.Fail()
	}
	value, e = p.GetUserRegisterNumber(8)
	if e != nil {
		t.Logf("Error reading user-bank r8: %s\n", e)
		t.Fail()
	}
	if value != 42 {
		t.Logf("User-mode r8 contained %d, not 42.\n", value)
		t.Fail()
	}
}

func TestBranchEmulation(t *testing.T) {
	p, e := setupTestProcessor()
	if e != nil {
		t.FailNow()
	}
	instructions := []uint32{
		// mov r0, 0
		0xe3a00000,
		// bl function
		0xeb000002,
		// add r0, r0, 1200
		0xe2800e4b,
		// part_2: mov r0, 137
		0xe3a00089,
		// mov pc, lr
		0xe1a0f00e,
		// function: cmp r0, 1
		0xe3500001,
		// bne part_2
		0x1afffffb,
		// mov r0, 200
		0xe3a000c8,
		// mov pc, lr
		0xe1a0f00e}

	e = writeInstructionsToMemory(instructions, p)
	if e != nil {
		t.FailNow()
	}
	p.SetRegisterNumber(15, 4096)
	e = runMultipleInstructions(7, p, t)
	if e != nil {
		t.Logf("Error running instructions: %s\n", e)
		t.Fail()
	}
	value, _ := p.GetRegisterNumber(0)
	if value != 1337 {
		t.Logf("Expected 1337 in r0, but got %d.\n", value)
		t.Fail()
	}
}

func TestCoprocessorEmulation(t *testing.T) {
	p, e := setupTestProcessor()
	if e != nil {
		t.FailNow()
	}
	e = p.AddCoprocessor(NewTestStorageCoprocessor(8))
	if e != nil {
		t.Logf("Couldn't add coprocessor: %s\n")
		t.FailNow()
	}
	instructions := []uint32{
		// mcr 8, 0, r0, ....
		0xee000810,
		// cdp 8, ....
		0xee000800,
		// stc 8, cr0, [r1], 4
		0xeca10801,
		// ldc 8, cr0, [r1]
		0xed910800,
		// mrc 8, 0, r0, ...
		0xee100810,
		// data...
		100, 200}
	e = writeInstructionsToMemory(instructions, p)
	if e != nil {
		t.FailNow()
	}
	p.SetRegisterNumber(15, 4096)
	p.SetRegisterNumber(0, 1336)
	dataStart := uint32(4096 + ((len(instructions) - 2) * 4))
	p.SetRegisterNumber(1, dataStart)
	e = runMultipleInstructions(5, p, t)
	if e != nil {
		t.Logf("Error running instructions: %s\n", e)
		t.Fail()
	}
	value, _ := p.GetRegisterNumber(1)
	if value != dataStart+4 {
		t.Logf("Incorrect value in r1. Expected %d, got %d.\n", dataStart+4,
			value)
		t.Fail()
	}
	value, _ = p.GetRegisterNumber(0)
	if value != 200 {
		t.Logf("Incorrect value in r0. Expected 200, got %d.\n", value)
		t.Fail()
	}
	value, e = p.GetMemoryInterface().ReadMemoryWord(dataStart)
	if e != nil {
		t.Logf("Memory read error at %08x: %s\n", dataStart, e)
		t.FailNow()
	}
	if value != 1337 {
		t.Logf("Expected 1337 in memory, found %d instead.\n", value)
		t.Fail()
	}
}

func TestSoftwareInterruptEmulation(t *testing.T) {
	p, e := setupTestProcessor()
	if e != nil {
		t.FailNow()
	}
	e = p.SetMode(0x10)
	if e != nil {
		t.Logf("Error putting processor in user mode: %s\n", e)
		t.FailNow()
	}
	// swi 0x1337
	e = testSingleInstruction(0xef001337, p)
	if e != nil {
		t.Logf("Error running swi instruction: %s\n", e)
		t.Fail()
	}
	newMode := p.GetMode()
	if newMode != 0x13 {
		t.Logf("Processor in mode 0x%02x, not 0x13, after swi.\n", newMode)
		t.Fail()
	}
	value, _ := p.GetRegisterNumber(15)
	if value != 0x8 {
		t.Logf("r15 set to 0x%08x, not 0x8, after swi.\n", value)
		t.Fail()
	}
	value, _ = p.GetRegisterNumber(14)
	if value != 4100 {
		t.Logf("r14 set to %d (expected 4100), after swi.\n", value)
		t.Fail()
	}
}
