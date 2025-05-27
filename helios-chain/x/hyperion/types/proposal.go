package types

import (
	"strings"

	errorsmod "cosmossdk.io/errors"
	v1beta1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1beta1"
)

// constants
const (
	ProposalHyperion string = "ProposalHyperion"
)

// Implements Proposal Interface
var (
	_ v1beta1.Content = &HyperionProposal{}
)

func init() {
	v1beta1.RegisterProposalType(ProposalHyperion)
}

/////////////////////////////////////////////////////////
// AddCounterpartyChainParamsProposal
/////////////////////////////////////////////////////////

func NewHyperionProposal(title, description, msg string) v1beta1.Content {
	return &HyperionProposal{
		Title:       title,
		Description: description,
		Msg:         msg,
	}
}

// ProposalRoute returns router key for this proposal
func (*HyperionProposal) ProposalRoute() string { return RouterKey }

// ProposalType returns proposal type for this proposal
func (*HyperionProposal) ProposalType() string {
	return ProposalHyperion
}

// ValidateBasic performs a stateless check of the proposal fields
func (p *HyperionProposal) ValidateBasic() error {
	// Validate title
	if strings.TrimSpace(p.Title) == "" {
		return errorsmod.Wrap(v1beta1.ErrInvalidLengthQuery, "proposal title cannot be empty")
	}

	// Validate description
	if strings.TrimSpace(p.Description) == "" {
		return errorsmod.Wrap(v1beta1.ErrInvalidLengthQuery, "proposal description cannot be empty")
	}

	return nil
}

// GetDescription returns the description of the proposal.
func (p *HyperionProposal) GetDescription() string {
	return p.Description
}

// GetTitle returns the title of the proposal.
func (p *HyperionProposal) GetTitle() string {
	return p.Title
}

/////////////////////////////////////////////////////////
