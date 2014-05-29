/*
The arm_emulate package contains utilities for disassembling and emulating
ARM bytecode.

A simple example of disassembling ARM and THUMB:

  arm_instruction, e := arm_emulate.ParseInstruction(0x00810002)
  if e == nil {
  	// Prints 'addeq r0, r1, r2'
  	fmt.Printf("%s\n", arm_instruction)
  } else {
  	fmt.Printf("Failed decoding instruction: %s\n", e)
  }

  thumb_instruction, e := arm_emulate.ParseTHUMBInstruction(0x1888)
  if e == nil {
  	// Prints 'add r0, r1, r2'
  	fmt.Printf("%s\n", thumb_instruction)
  } else {
  	fmt.Printf("Failed decoding instruction: %s\n", e)
  }

Type assertions may be used to convert objects returned by ParseInstruction or
ParseTHUMBInstruction into more specific instruction types, through which
fields specific to the instruction may be accessed.

An example of emulating instructions:

  processor := arm_emulate.NewARMProcessor()
  memory := processor.GetMemoryInterface()
  // The memory interface starts out in little endian mode, so this is an add
  // instruction.
  codeBytes := []byte{0x02, 0x00, 0x81, 0xe0}
  // This allocates memory starting at address 4096, and copies the bytes into
  // it.
  memory.SetMemoryRegion(4096, codeBytes)
  // Set the PC to point to the start of the memory we just mapped
  processor.SetRegisterNumber(15, 4096)
  // We can initialize any other register in a similar manner
  processor.SetRegisterNumber(1, 1234)
  var e error
  // This will emulate instructions in a loop until an error occurs.
  for e == nil {
  	e = processor.RunNextInstruction()
  }
  fmt.Printf("Emulation ended due to an error: %s\n")

After the processor, memory and PC have been initialized, the
RunNextInstruction() function of the ARMProcessor may be called in a loop to
carry out emulation. It returns an error when one occurs.
*/
package arm_emulate
