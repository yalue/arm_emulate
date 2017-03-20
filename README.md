ARM Emulation Framework for Go
==================================

About
-----
This project implements an ARM dissassembly and emulation library for the Go
programming language. The initial version is based on ARM7, so features added
in later ARM versions may not be supported. Even so, support for the basic ARM
and THUMB instruction sets is complete.

Usage
-----
First, download the code: `go get -v github.com/yalue/arm_emulate`

The arm\_emulate package includes both disassembly and emulation functions.

A simple function to disassemble an ARM instruction:
```go
package main

import (
    "fmt"
    "github.com/yalue/arm_emulate"
)

func main() {
    // 0xe0810002 is the binary encoding of 'add r0, r1, r2'
    arm_instruction, e := arm_emulate.ParseInstruction(0xe0810002)
    if e != nil {
        fmt.Printf("Error disassembling: %s\n", e)
    } else {
        // This will print 'add r0, r1, r2'
        fmt.Printf("%s\n", arm_instruction)
    }
}
```
The primary functions of interest for disassembly probably will be
`ParseInstruction` and `ParseTHUMBInstruction` which take 32 or 16-bit values
and return ARMInstruction or THUMBInstruction values, respectively. If these
functions don't return an error, type assertions can be used to convert the
values returned into a specific instruction type, through which individual
fields, such as registers or immediate values, may be accessed.

An example of emulating instructions:
```go
package main

import (
    "fmt"
    "github.com/yalue/arm_emulate"
)

func main() {
    // First, create a processor. The 'memory' variable doesn't need to be
    // created, but it helps keep code shorter.
    processor := arm_emulate.NewARMProcessor()
    memory := processor.GetMemoryInterface()

    // The memory interface starts out in little endian mode, so this is an add
    // instruction, a mov instruction, and an undefined instruction.
    codeBytes := []byte{0x02, 0x00, 0x81, 0xe0, 0x01, 0x00, 0x81, 0xe0, 0xff,
        0xff, 0xff, 0x07}

    // This allocates memory starting at address 4096, and copies the bytes
    // into it.
    memory.SetMemoryRegion(4096, codeBytes)

    // Set the PC to point to the start of the memory we just mapped
    processor.SetRegister(15, 4096)

    // We can initialize any other register in a similar manner
    processor.SetRegister(1, 1234)

    var e error
    // This will emulate instructions in a loop until an error occurs.
    for e == nil {
        // Print a trace of the instructions running
        fmt.Printf("%s\n", processor.PendingInstructionString())
        e = processor.RunNextInstruction()
    }
    fmt.Printf("Emulation ended due to an error: %s\n", e)
}
```

Coprocessors may be implemented using the ARMCoprocessor interface. See the
coprocessor.go file for this definition and an implementation of a simple
counter coprocessor. The usage of this can be seen in the emulate_test.go file,
in the TestCoprocessorEmulation test case.

Further documentation, including a complete list of types, can be found in the
go documentation for the arm_emulate package.

Coding and Naming Conventions
-----------------------------
Aside from following the Go guidelines and 80-character lines, a few
conventions were used throughout this project. Instruction types were based on
the formats given in the ARM7TDMI instruction set reference (for ARM and
THUMB). Numerical fields from instructions are always stored in the shortest
possible type that can hold them (for example, an 11 bit integer is stored in a
16-bit type rather than 32-bit). Also, the instruction types always use\
unsigned values, without shifts or rotates applied--their sole purpose is to
split the instruction into its fields, not to carry out evaluation.

Names for each instruction type is based on the name given to the format in
the aforementioned documents. If the same name appears in both ARM and THUMB
formats, the ARM version of the type uses the name as it is, and the THUMB
version has 'THUMB' appended to the name.

When writing test cases, ARM bytecode is kept directly in the go test files,
as slices of bytes, uint16s (halfwords) and uint32s (words) where appropriate.

Planned Features
----------------

 - Implement the MMU coprocessor (for the sake of better ARM9 support)

 - Implement the VFP (floating point) coprocessor

 - Support the Thumb2 extensions

 - (Long term) Support 64-bit (ARMv8) features

