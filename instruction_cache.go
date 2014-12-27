package arm_emulate

// TODO: These functions will introduce race conditions if used in a multicore
// context.

// This file aims to cache allocated instructions so that they won't need to be
// reallocated every time they are parsed. In a single-threaded emulator, this
// can save many allocations, which need to happen at a high speed. In order to
// enable this to work, the user of the instruction should call the ReCache()
// method once the instruction is no longer in use.

var cachedDataProcessingInstruction *DataProcessingInstruction

func (n *DataProcessingInstruction) ReCache() {
	cachedDataProcessingInstruction = n
}

func newDataProcessingInstruction() *DataProcessingInstruction {
	var toReturn *DataProcessingInstruction
	if cachedDataProcessingInstruction != nil {
		toReturn = cachedDataProcessingInstruction
		cachedDataProcessingInstruction = nil
	} else {
		toReturn = &DataProcessingInstruction{}
	}
	return toReturn
}

var cachedSingleDataTransferInstruction *SingleDataTransferInstruction

func (n *SingleDataTransferInstruction) ReCache() {
	cachedSingleDataTransferInstruction = n
}

func newSingleDataTransferInstruction() *SingleDataTransferInstruction {
	var toReturn *SingleDataTransferInstruction
	if cachedSingleDataTransferInstruction != nil {
		toReturn = cachedSingleDataTransferInstruction
		cachedSingleDataTransferInstruction = nil
	} else {
		toReturn = &SingleDataTransferInstruction{}
	}
	return toReturn
}

var cachedBranchInstruction *BranchInstruction

func (n *BranchInstruction) ReCache() {
	cachedBranchInstruction = n
}

func newBranchInstruction() *BranchInstruction {
	var toReturn *BranchInstruction
	if cachedBranchInstruction != nil {
		toReturn = cachedBranchInstruction
		cachedBranchInstruction = nil
	} else {
		toReturn = &BranchInstruction{}
	}
	return toReturn
}

var cachedBranchExchangeInstruction *BranchExchangeInstruction

func (n *BranchExchangeInstruction) ReCache() {
	cachedBranchExchangeInstruction = n
}

func newBranchExchangeInstruction() *BranchExchangeInstruction {
	var toReturn *BranchExchangeInstruction
	if cachedBranchExchangeInstruction != nil {
		toReturn = cachedBranchExchangeInstruction
		cachedBranchExchangeInstruction = nil
	} else {
		toReturn = &BranchExchangeInstruction{}
	}
	return toReturn
}
