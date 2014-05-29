package arm_emulate

import (
	"fmt"
)

// This defines the interface to memory to be used during emulation.
type ARMMemory interface {
	// Maps the given byte array into memory, starting at the given base
	// address. baseAddress doesn't need to be aligned with anything.
	SetMemoryRegion(baseAddress uint32, memory []byte) error
	// "Unmaps" the given range. Size is rounded down to the nearest 4096 bytes
	// and baseAddress is page-aligned. This can free unused memory.
	ClearMemoryRegion(baseAddress uint32, size uint32) error
	ReadMemoryWord(address uint32) (uint32, error)
	WriteMemoryWord(address uint32, data uint32) error
	ReadMemoryHalfword(address uint32) (uint16, error)
	WriteMemoryHalfword(address uint32, data uint16) error
	ReadMemoryByte(address uint32) (uint8, error)
	WriteMemoryByte(address uint32, data uint8) error
	// Endianness only matters when reading or writing words or halfwords.
	// Values used in the interface (addresses and values) should always use
	// the "native" endianness of the machine running the emulator.
	SetBigEndian(bigEndian bool)
	IsBigEndian() bool
}

// Uses 2-level page tables and 4k pages.
type basicARMMemory struct {
	pages       [][][]byte
	isBigEndian bool
}

// Returns the 2nd-level index, page table index, and offset, respectively.
func getAddressPageIndices(address uint32) (uint32, uint32, uint32) {
	return address >> 20, (address >> 12) & 0xff, address & 0xfff
}

// Returns the page containing the given address, or an error if the page
// didn't already exist.
func (m *basicARMMemory) getContainingPage(address uint32) ([]byte, error) {
	level2Index, level1Index, _ := getAddressPageIndices(address)
	level1Table := m.pages[level2Index]
	if level1Table == nil {
		return nil, fmt.Errorf("Page doesn't exist: 0x%08x", address)
	}
	page := level1Table[level1Index]
	if page == nil {
		return nil, fmt.Errorf("Page doesn't exist: 0x%08x", address)
	}
	return page, nil
}

// Like getContainingPage, but creates the page if it doesn't exist.
func (m *basicARMMemory) createContainingPage(address uint32) []byte {
	level2Index, level1Index, _ := getAddressPageIndices(address)
	if m.pages[level2Index] == nil {
		m.pages[level2Index] = make([][]byte, 256)
	}
	level1Table := m.pages[level2Index]
	if level1Table[level1Index] == nil {
		level1Table[level1Index] = make([]byte, 4096)
	}
	return level1Table[level1Index]
}

func (m *basicARMMemory) ReadMemoryWord(address uint32) (uint32, error) {
	var toReturn uint32
	address &= 0xfffffffc
	page, e := m.getContainingPage(address)
	if e != nil {
		return 0, e
	}
	offset := int(address & 0xfff)
	if m.isBigEndian {
		for i := 0; i < 4; i++ {
			toReturn = toReturn << 8
			toReturn |= uint32(page[offset+i])
		}
	} else {
		for i := int(3); i >= 0; i-- {
			toReturn = toReturn << 8
			toReturn |= uint32(page[offset+i])
		}
	}
	return toReturn, nil
}

func (m *basicARMMemory) WriteMemoryWord(address uint32, value uint32) error {
	address &= 0xfffffffc
	page, e := m.getContainingPage(address)
	if e != nil {
		return e
	}
	offset := int(address & 0xfff)
	if m.isBigEndian {
		for i := 0; i < 4; i++ {
			page[offset+i] = byte((value & 0xff000000) >> 24)
			value = value << 8
		}
	} else {
		for i := 0; i < 4; i++ {
			page[offset+i] = byte(value & 0xff)
			value = value >> 8
		}
	}
	return nil
}

func (m *basicARMMemory) ReadMemoryHalfword(address uint32) (uint16, error) {
	address &= 0xfffffffe
	page, e := m.getContainingPage(address)
	if e != nil {
		return 0, e
	}
	offset := address & 0xfff
	if m.isBigEndian {
		return (uint16(page[offset]) << 8) | uint16(page[offset+1]), nil
	}
	return (uint16(page[offset+1]) << 8) | uint16(page[offset]), nil
}

func (m *basicARMMemory) WriteMemoryHalfword(address uint32,
	data uint16) error {
	address &= 0xfffffffe
	page, e := m.getContainingPage(address)
	if e != nil {
		return e
	}
	offset := address & 0xfff
	if m.isBigEndian {
		page[offset] = byte((data & 0xff00) >> 8)
		page[offset+1] = byte(data & 0xff)
	} else {
		page[offset] = byte(data & 0xff)
		page[offset+1] = byte((data & 0xff00) >> 8)
	}
	return nil
}

func (m *basicARMMemory) ReadMemoryByte(address uint32) (uint8, error) {
	page, e := m.getContainingPage(address)
	if e != nil {
		return 0, e
	}
	return page[address&0xfff], nil
}

func (m *basicARMMemory) WriteMemoryByte(address uint32, value uint8) error {
	page, e := m.getContainingPage(address)
	if e != nil {
		return e
	}
	page[address&0xfff] = value
	return nil
}

func (m *basicARMMemory) SetMemoryRegion(baseAddress uint32,
	memory []byte) error {
	if (uint64(baseAddress) + uint64(len(memory))) > uint64(0xffffffff) {
		return fmt.Errorf("Not enough space to map %d bytes at 0x%08x",
			len(memory), baseAddress)
	}
	address := baseAddress & 0xfffff000
	offset := baseAddress & 0xfff
	page := m.createContainingPage(address)
	for i := 0; i < len(memory); i++ {
		page[offset] = memory[i]
		offset++
		if offset >= 4096 {
			offset = 0
			address += 4096
			page = m.createContainingPage(address)
		}
	}
	return nil
}

func (m *basicARMMemory) ClearMemoryRegion(baseAddress uint32,
	size uint32) error {
	address := baseAddress
	if (address % 4096) != 0 {
		address += 4096 - (address % 4096)
	}
	limitAddress := (baseAddress + size) & 0xfffff000
	// Free pages
	for address < limitAddress {
		level2Index, level1Index, _ := getAddressPageIndices(address)
		m.pages[level2Index][level1Index] = nil
		address += 4096
	}
	address = baseAddress & 0xfff00000
	if (address % 0x100000) != 0 {
		address += 0x100000 - (address % 0x100000)
	}
	limitAddress = (baseAddress + size) & 0xfff00000
	// Free page tables
	for address < limitAddress {
		level2Index, _, _ := getAddressPageIndices(address)
		m.pages[level2Index] = nil
		address += 0x100000
	}
	return nil
}

func (m *basicARMMemory) SetBigEndian(isBigEndian bool) {
	m.isBigEndian = isBigEndian
}

func (m *basicARMMemory) IsBigEndian() bool {
	return m.isBigEndian
}

// Returns a new ARMMemory object, empty and set to little endian
func NewARMMemory() ARMMemory {
	var toReturn basicARMMemory
	// The top-level page table of 1MB pages, for 4GB possible memory
	toReturn.pages = make([][][]byte, 4096)
	toReturn.isBigEndian = false
	return &toReturn
}
