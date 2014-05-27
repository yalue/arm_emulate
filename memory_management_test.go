package arm_emulate

import (
	"testing"
)

func TestEmptyMemory(t *testing.T) {
	m := NewARMMemory()
	_, e := m.ReadMemoryByte(0x10)
	if e == nil {
		t.Fail()
	}
	_, e = m.ReadMemoryHalfword(0x10)
	if e == nil {
		t.Fail()
	}
	_, e = m.ReadMemoryWord(0x10)
	if e == nil {
		t.Fail()
	}
	e = m.WriteMemoryByte(0x10, 1)
	if e == nil {
		t.Fail()
	}
	e = m.WriteMemoryHalfword(0x10, 1)
	if e == nil {
		t.Fail()
	}
	e = m.WriteMemoryWord(0x10, 1)
	if e == nil {
		t.Fail()
	}
}

func TestInsufficientSpace(t *testing.T) {
	data := []byte{0x12, 0x34, 0x56, 0x78}
	m := NewARMMemory()
	// Test not enough space
	e := m.SetMemoryRegion(0xffffffff, data)
	if e == nil {
		t.Fail()
	}
	_, e = m.ReadMemoryByte(0xffffffff)
	if e == nil {
		t.Fail()
	}
}

func TestReadMemory(t *testing.T) {
	data := []byte{0x12, 0x34, 0x56, 0x78}
	m := NewARMMemory()
	e := m.SetMemoryRegion(0x10, data)
	if e != nil {
		t.Fail()
	}
	b, e := m.ReadMemoryByte(0x10)
	if e != nil {
		t.Fail()
	}
	if b != 0x12 {
		t.Fail()
	}
	b, e = m.ReadMemoryByte(0x12)
	if e != nil {
		t.Fail()
	}
	if b != 0x56 {
		t.Fail()
	}
	h, e := m.ReadMemoryHalfword(0x10)
	if e != nil {
		t.Fail()
	}
	if h != 0x3412 {
		t.Fail()
	}
	h, e = m.ReadMemoryHalfword(0x11)
	if e != nil {
		t.Fail()
	}
	if h != 0x3412 {
		t.Fail()
	}
	w, e := m.ReadMemoryWord(0x10)
	if e != nil {
		t.Fail()
	}
	if w != 0x78563412 {
		t.Fail()
	}
}

func TestWriteMemory(t *testing.T) {
	data := make([]byte, 0x100000)
	m := NewARMMemory()
	if m.SetMemoryRegion(0x10000000, data) != nil {
		t.Fail()
	}
	if m.WriteMemoryByte(0x10000000, 0x13) != nil {
		t.Fail()
	}
	if m.WriteMemoryByte(0x10000001, 0x37) != nil {
		t.Fail()
	}
	w, e := m.ReadMemoryWord(0x10000000)
	if e != nil {
		t.Fail()
	}
	if w != 0x00003713 {
		t.Fail()
	}
	if m.WriteMemoryHalfword(0x10002000, 0x1337) != nil {
		t.Fail()
	}
	w, e = m.ReadMemoryWord(0x10002000)
	if e != nil {
		t.Fail()
	}
	if w != 0x00001337 {
		t.Fail()
	}
	if m.WriteMemoryWord(0x10004000, 0x13371337) != nil {
		t.Fail()
	}
	w, e = m.ReadMemoryWord(0x10004000)
	if e != nil {
		t.Fail()
	}
	if w != 0x13371337 {
		t.Fail()
	}
	b, e := m.ReadMemoryByte(0x10004000)
	if e != nil {
		t.Fail()
	}
	if b != 0x37 {
		t.Fail()
	}
}

func TestBigEndian(t *testing.T) {
	data := []byte{1, 2, 3, 4}
	m := NewARMMemory()
	if m.IsBigEndian() {
		t.Fail()
	}
	if m.SetMemoryRegion(0x10, data) != nil {
		t.Fail()
	}
	w, e := m.ReadMemoryWord(0x10)
	if e != nil {
		t.Fail()
	}
	if w != 0x04030201 {
		t.Fail()
	}
	m.SetBigEndian(true)
	if !m.IsBigEndian() {
		t.Fail()
	}
	w, e = m.ReadMemoryWord(0x10)
	if e != nil {
		t.Fail()
	}
	if w != 0x01020304 {
		t.Fail()
	}
	if m.WriteMemoryHalfword(0x12, 0x1337) != nil {
		t.Fail()
	}
	w, e = m.ReadMemoryWord(0x10)
	if e != nil {
		t.Fail()
	}
	if w != 0x01021337 {
		t.Fail()
	}
	m.SetBigEndian(false)
	if m.IsBigEndian() {
		t.Fail()
	}
	w, e = m.ReadMemoryWord(0x10)
	if e != nil {
		t.Fail()
	}
	if w != 0x37130201 {
		t.Fail()
	}
}

func TestClearMemoryRegion(t *testing.T) {
	data := make([]byte, 4096)
	m := NewARMMemory()
	_, e := m.ReadMemoryByte(4096)
	if e == nil {
		t.Fail()
	}
	// This should map the first two pages, since the data overlaps both
	if m.SetMemoryRegion(2048, data) != nil {
		t.Fail()
	}
	_, e = m.ReadMemoryByte(0)
	if e != nil {
		println("Didn't map first overlapped page.")
		t.Fail()
	}
	_, e = m.ReadMemoryByte(5000)
	if e != nil {
		println("Didn't map second overlapped page.")
		t.Fail()
	}
	// Shouldn't do anything, page to clear are rounded down
	if m.ClearMemoryRegion(0, 4000) != nil {
		t.Fail()
	}
	_, e = m.ReadMemoryByte(0)
	if e != nil {
		println("Unmapped when too few bytes were cleared.")
		t.Fail()
	}
	// Shouldn't do anything, these were never mapped
	if m.ClearMemoryRegion(0x10000, 0x10000) != nil {
		t.Fail()
	}
	// Also shouldn't do anything, doesn't cover any single entire page
	if m.ClearMemoryRegion(2048, 5000) != nil {
		t.Fail()
	}
	_, e = m.ReadMemoryByte(0)
	if e != nil {
		println("Unmapped second page when only partially cleared.")
		t.Fail()
	}
	_, e = m.ReadMemoryByte(4096)
	if e != nil {
		println("Unmapped first page when only partially cleared.")
		t.Fail()
	}
	// This should unmap the second page only
	if m.ClearMemoryRegion(4096, 4096) != nil {
		t.Fail()
	}
	_, e = m.ReadMemoryByte(4096)
	if e == nil {
		println("Was able to read from unmapped page.")
		t.Fail()
	}
	_, e = m.ReadMemoryByte(0)
	if e != nil {
		println("Unmapped first page along with second.")
		t.Fail()
	}
}
