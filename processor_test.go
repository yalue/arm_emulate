package arm_emulate

import (
	"testing"
)

func TestIRQDelivery(t *testing.T) {
	p, e := setupTestProcessor()
	if e != nil {
		t.FailNow()
	}
	p.SetRegisterNumber(15, 4096)
	e = p.SendIRQ()
	if e != nil {
		t.Logf("Got an error sending an IRQ: %s\n", e)
		t.Fail()
	}
	value, _ := p.GetRegisterNumber(15)
	if value != 0x18 {
		t.Logf("The processor wasn't at the IRQ vector, but 0x%08x.\n", value)
		t.Fail()
	}
	value, _ = p.GetRegisterNumber(14)
	if value != 4100 {
		t.Logf("IRQ has wrong return address: %d instead of 4100.\n", value)
		t.Fail()
	}
	if p.GetMode() != 0x12 {
		t.Logf("Processor failed to enter IRQ mode. Mode is 0x%02x instead.\n",
			p.GetMode())
		t.Fail()
	}
	p.SetRegisterNumber(15, 4096)
	status, e := p.GetCPSR()
	if e != nil {
		t.Logf("Failed getting CPSR in IRQ test.\n")
		t.Fail()
	}
	status |= 1 << 7
	e = p.SetCPSR(status)
	if e != nil {
		t.Logf("Failed setting the IRQ disable bit: %s\n", e)
		t.Fail()
	}
	e = p.SendIRQ()
	if e != nil {
		t.Logf("Error sending an IRQ with IRQ disabled: %s\n", e)
		t.Fail()
	}
	value, _ = p.GetRegisterNumber(15)
	if value != 4096 {
		t.Logf("IRQ modified the PC with IRQs disabled.\n")
		t.Fail()
	}
}

func TestFIQDelivery(t *testing.T) {
	p, e := setupTestProcessor()
	if e != nil {
		t.FailNow()
	}
	p.SetRegisterNumber(15, 4096)
	e = p.SendFIQ()
	if e != nil {
		t.Logf("Got an error sending an FIQ: %s\n", e)
		t.Fail()
	}
	value, _ := p.GetRegisterNumber(15)
	if value != 0x1c {
		t.Logf("The processor wasn't at the FIQ vector, but 0x%08x.\n", value)
		t.Fail()
	}
	value, _ = p.GetRegisterNumber(14)
	if value != 4100 {
		t.Logf("FIQ has wrong return address: %d instead of 4100.\n", value)
		t.Fail()
	}
	if p.GetMode() != 0x11 {
		t.Logf("Processor failed to enter FIQ mode. Mode is 0x%02x instead.\n",
			p.GetMode())
		t.Fail()
	}
	p.SetRegisterNumber(15, 4096)
	status, e := p.GetCPSR()
	if e != nil {
		t.Logf("Failed getting CPSR in FIQ test.\n")
		t.Fail()
	}
	status |= 1 << 6
	e = p.SetCPSR(status)
	if e != nil {
		t.Logf("Failed setting the FIQ disable bit: %s\n", e)
		t.Fail()
	}
	e = p.SendFIQ()
	if e != nil {
		t.Logf("Error sending an FIQ with FIQ disabled: %s\n", e)
		t.Fail()
	}
	value, _ = p.GetRegisterNumber(15)
	if value != 4096 {
		t.Logf("FIQ modified the PC with FIQs disabled.\n")
		t.Fail()
	}
}
