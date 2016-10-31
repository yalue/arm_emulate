package arm_emulate

// This file implements a basic 2-way set associative cache for ARM instructions

const (
	cacheSets = 64
)

// Holds 2 cache "ways" for ARM instructions, as well as an indicator for which
// was used most recently.
type armInstructionCacheSet struct {
	first         ARMInstruction
	second        ARMInstruction
	firstUsedLast bool
}

// Holds 2 cache ways for THUMB instructions.
type thumbInstructionCacheSet struct {
	first         THUMBInstruction
	second        THUMBInstruction
	firstUsedLast bool
}

// The top-level instruction cache for ARM and THUMB.
type instructionCache struct {
	armInstructions   []armInstructionCacheSet
	thumbInstructions []thumbInstructionCacheSet
}

func hashARMInstruction(raw uint32) uint32 {
	return (raw ^ (raw >> 27)) % cacheSets
}

func hashTHUMBInstruction(raw uint16) uint16 {
	return (raw ^ (raw >> 8)) % cacheSets
}

// Gets the ARM instruction at the given cache. Returns nil if the instruction
// cached.
func (c *instructionCache) getARMInstruction(raw uint32) ARMInstruction {
	set := &(c.armInstructions[hashARMInstruction(raw)])
	if set.first == nil {
		return nil
	}
	if set.first.Raw() == raw {
		set.firstUsedLast = true
		return set.first
	}
	if set.second == nil {
		return nil
	}
	if set.second.Raw() != raw {
		return nil
	}
	set.firstUsedLast = false
	return set.second
}

func (c *instructionCache) getTHUMBInstruction(raw uint16) THUMBInstruction {
	set := &(c.thumbInstructions[hashTHUMBInstruction(raw)])
	if set.first == nil {
		return nil
	}
	if set.first.Raw() == raw {
		set.firstUsedLast = true
		return set.first
	}
	if set.second == nil {
		return nil
	}
	if set.second.Raw() != raw {
		return nil
	}
	set.firstUsedLast = false
	return set.second
}

func (c *instructionCache) storeARMInstruction(n ARMInstruction) {
	set := &(c.armInstructions[hashARMInstruction(n.Raw())])
	if set.first == nil {
		set.first = n
		set.firstUsedLast = true
		return
	}
	if set.second == nil {
		set.second = n
		set.firstUsedLast = false
		return
	}
	if set.firstUsedLast {
		set.second = n
	} else {
		set.first = n
	}
	set.firstUsedLast = !set.firstUsedLast
}

func (c *instructionCache) storeTHUMBInstruction(n THUMBInstruction) {
	set := &(c.thumbInstructions[hashTHUMBInstruction(n.Raw())])
	if set.first == nil {
		set.first = n
		set.firstUsedLast = true
		return
	}
	if set.second == nil {
		set.second = n
		set.firstUsedLast = false
		return
	}
	if set.firstUsedLast {
		set.second = n
	} else {
		set.first = n
	}
	set.firstUsedLast = !set.firstUsedLast
}

func newInstructionCache() *instructionCache {
	var toReturn instructionCache
	toReturn.armInstructions = make([]armInstructionCacheSet, cacheSets)
	toReturn.thumbInstructions = make([]thumbInstructionCacheSet, cacheSets)
	return &toReturn
}
