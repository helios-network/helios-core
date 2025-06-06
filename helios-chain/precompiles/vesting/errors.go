package vesting

const (
	// ErrDifferentFromOrigin is raised when the tx origin address is not the same as the vesting transaction initiator.
	ErrDifferentFromOrigin = "tx origin address %s does not match the from address %s"
	// ErrDifferentFunderOrigin is raised when the tx origin address is not the same as the vesting transaction funder.
	ErrDifferentFunderOrigin = "tx origin address %s does not match the funder address %s"
	// ErrInvalidDestination is raised when the destination address is not an EOA or the contract calling the precompile.
	ErrInvalidDestination = "invalid destination address %s. Should be an EOA or the contract calling the precompile"
)
