// Copyright Tharsis Labs Ltd.(Evmos)
// SPDX-License-Identifier:ENCL-1.0(https://github.com/evmos/evmos/blob/main/LICENSE)

package types

import (
	"errors"
	"fmt"
	"strings"

	evmostypes "helios-core/helios-chain/types"

	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	v1beta1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1beta1"
)

// constants
const (
	// ProposalTypeRegisterCoin is DEPRECATED, remove after v16 upgrade
	ProposalTypeRegisterCoin          string = "RegisterCoin"
	ProposalTypeRegisterERC20         string = "RegisterERC20"
	ProposalTypeToggleTokenConversion string = "ToggleTokenConversion" // #nosec
	ProposalAddNewAssetConsensus      string = "AddNewAssetConsensus"
	ProposalRemoveAssetConsensus      string = "RemoveAssetConsensus"
	ProposalUpdateAssetConsensus      string = "UpdateAssetConsensus"
)

// Implements Proposal Interface
var (
	_ v1beta1.Content = &RegisterCoinProposal{}
	_ v1beta1.Content = &RegisterERC20Proposal{}
	_ v1beta1.Content = &ToggleTokenConversionProposal{}
	_ v1beta1.Content = &AddNewAssetConsensusProposal{}
	_ v1beta1.Content = &RemoveAssetConsensusProposal{}
	_ v1beta1.Content = &UpdateAssetConsensusProposal{}
)

func init() {
	v1beta1.RegisterProposalType(ProposalTypeRegisterERC20)
	v1beta1.RegisterProposalType(ProposalTypeToggleTokenConversion)
	v1beta1.RegisterProposalType(ProposalAddNewAssetConsensus)
	v1beta1.RegisterProposalType(ProposalRemoveAssetConsensus)
	v1beta1.RegisterProposalType(ProposalUpdateAssetConsensus)
}

func (rtbp *RegisterCoinProposal) ValidateBasic() error {
	return errors.New("deprecated")
}

// ValidateBasic performs a stateless check of the proposal fields
func (rtbp *RegisterERC20Proposal) ValidateBasic() error {
	for _, address := range rtbp.Erc20Addresses {
		if err := evmostypes.ValidateAddress(address); err != nil {
			return errorsmod.Wrap(err, "ERC20 address")
		}
	}

	return v1beta1.ValidateAbstract(rtbp)
}

// CreateDenomDescription generates a string with the coin description
func CreateDenomDescription(address string) string {
	return fmt.Sprintf("Cosmos coin token representation of %s", address)
}

// CreateDenom generates a string the module name plus the address to avoid conflicts with names starting with a number
func CreateDenom(address string) string {
	return fmt.Sprintf("%s/%s", ModuleName, address)
}

// ValidateErc20Denom checks if a denom is a valid erc20/ denomination
func ValidateErc20Denom(denom string) error {
	denomSplit := strings.SplitN(denom, "/", 2)

	if len(denomSplit) != 2 || denomSplit[0] != ModuleName {
		return fmt.Errorf("invalid denom. %s denomination should be prefixed with the format 'erc20/", denom)
	}

	return evmostypes.ValidateAddress(denomSplit[1])
}

// NewRemoveAssetConsensusProposal creates a new proposal to remove assets from the whitelist
func NewRemoveAssetConsensusProposal(title, description string, denoms []string) v1beta1.Content {
	return &RemoveAssetConsensusProposal{
		Title:       title,
		Description: description,
		Denoms:      denoms,
	}
}

// ProposalRoute returns router key for this proposal
func (*RemoveAssetConsensusProposal) ProposalRoute() string { return RouterKey }

// ProposalType returns proposal type for this proposal
func (*RemoveAssetConsensusProposal) ProposalType() string {
	return ProposalRemoveAssetConsensus
}

// ValidateBasic performs a stateless check of the proposal fields
func (p *RemoveAssetConsensusProposal) ValidateBasic() error {
	// Validate title
	if strings.TrimSpace(p.Title) == "" {
		return errorsmod.Wrap(v1beta1.ErrInvalidLengthQuery, "proposal title cannot be empty")
	}

	// Validate description
	if strings.TrimSpace(p.Description) == "" {
		return errorsmod.Wrap(v1beta1.ErrInvalidLengthQuery, "proposal description cannot be empty")
	}

	// Validate denoms
	if len(p.Denoms) == 0 {
		return errorsmod.Wrap(v1beta1.ErrInvalidLengthQuery, "proposal must include at least one denom")
	}

	for _, denom := range p.Denoms {
		if strings.TrimSpace(denom) == "" {
			return errorsmod.Wrap(v1beta1.ErrInvalidLengthQuery, "denom cannot be empty")
		}
	}

	return nil
}

// GetDescription returns the description of the proposal.
func (p *RemoveAssetConsensusProposal) GetDescription() string {
	return p.Description
}

// GetTitle returns the title of the proposal.
func (p *RemoveAssetConsensusProposal) GetTitle() string {
	return p.Title
}

// ProposalRoute returns router key for this proposal.
func (*ToggleTokenConversionProposal) ProposalRoute() string { return RouterKey }

// ProposalType returns proposal type for this proposal
func (*ToggleTokenConversionProposal) ProposalType() string {
	return ProposalTypeToggleTokenConversion
}

// ValidateBasic performs a stateless check of the proposal fields
func (ttcp *ToggleTokenConversionProposal) ValidateBasic() error {
	// check if the token is a hex address, if not, check if it is a valid SDK
	// denom
	if err := evmostypes.ValidateAddress(ttcp.Token); err != nil {
		if err := sdk.ValidateDenom(ttcp.Token); err != nil {
			return err
		}
	}

	return v1beta1.ValidateAbstract(ttcp)
}

// ProposalRoute returns router key of this proposal.
func (p *AddNewAssetConsensusProposal) ProposalRoute() string { return RouterKey }

// ProposalType returns proposal type of this proposal.
func (p *AddNewAssetConsensusProposal) ProposalType() string {
	return ProposalAddNewAssetConsensus
}

// ValidateBasic performs a stateless check of the proposal fields.
func (p *AddNewAssetConsensusProposal) ValidateBasic() error {
	// Validate title
	if strings.TrimSpace(p.Title) == "" {
		return errorsmod.Wrap(v1beta1.ErrInvalidLengthQuery, "proposal title cannot be empty")
	}

	// Validate description
	if strings.TrimSpace(p.Description) == "" {
		return errorsmod.Wrap(v1beta1.ErrInvalidLengthQuery, "proposal description cannot be empty")
	}

	// Validate assets
	if len(p.Assets) == 0 {
		return errorsmod.Wrap(v1beta1.ErrInvalidLengthQuery, "proposal must include at least one asset")
	}

	for _, asset := range p.Assets {
		// Validate asset denom
		if strings.TrimSpace(asset.Denom) == "" {
			return errorsmod.Wrap(v1beta1.ErrInvalidLengthQuery, "asset denom cannot be empty")
		}

		// Validate contract address
		if strings.TrimSpace(asset.ContractAddress) == "" {
			return errorsmod.Wrap(v1beta1.ErrInvalidLengthQuery, "asset contract address cannot be empty")
		}

		// TODO: link with hyperion to know the list of authorized chains
		// Validate chain ID
		if strings.TrimSpace(asset.ChainId) == "" {
			return errorsmod.Wrap(v1beta1.ErrInvalidLengthQuery, "asset chain ID cannot be empty")
		}

		// Validate decimals
		if asset.Decimals == 0 {
			return errorsmod.Wrap(v1beta1.ErrInvalidLengthQuery, "asset decimals must be greater than zero")
		}

		// Validate base weight
		if asset.BaseWeight == 0 {
			return errorsmod.Wrap(v1beta1.ErrInvalidLengthQuery, "asset base weight must be greater than zero")
		}
	}

	return nil
}

// GetDescription returns the description of this proposal.
func (p *AddNewAssetConsensusProposal) GetDescription() string {
	return p.Description
}

// GetTitle returns the title of this proposal.
func (p *AddNewAssetConsensusProposal) GetTitle() string {
	return p.Title
}

// ProposalRoute returns the router key for this proposal.
func (*RegisterCoinProposal) ProposalRoute() string {
	return RouterKey
}

// ProposalType returns the proposal type for this proposal.
func (*RegisterCoinProposal) ProposalType() string {
	return ProposalTypeRegisterCoin
}

// ProposalRoute returns the router key for this proposal.
func (*RegisterERC20Proposal) ProposalRoute() string {
	return RouterKey
}

// ProposalType returns the proposal type for this proposal.
func (*RegisterERC20Proposal) ProposalType() string {
	return ProposalTypeRegisterERC20
}

// GetDescription returns the description of this proposal.
func (p *UpdateAssetConsensusProposal) GetDescription() string {
	return p.Description
}

// GetTitle returns the title of the proposal.
func (p *UpdateAssetConsensusProposal) GetTitle() string {
	return p.Title
}

// ProposalRoute returns the router key for this proposal.
func (*UpdateAssetConsensusProposal) ProposalRoute() string {
	return RouterKey
}

// ProposalType returns the proposal type for this proposal.
func (*UpdateAssetConsensusProposal) ProposalType() string {
	return ProposalUpdateAssetConsensus
}

// ValidateBasic performs a stateless check of the proposal fields.
func (p *UpdateAssetConsensusProposal) ValidateBasic() error {
	// Validate title
	if strings.TrimSpace(p.Title) == "" {
		return errorsmod.Wrap(v1beta1.ErrInvalidLengthQuery, "proposal title cannot be empty")
	}

	// Validate description
	if strings.TrimSpace(p.Description) == "" {
		return errorsmod.Wrap(v1beta1.ErrInvalidLengthQuery, "proposal description cannot be empty")
	}

	// Validate updates
	if len(p.Updates) == 0 {
		return errorsmod.Wrap(v1beta1.ErrInvalidLengthQuery, "proposal must include at least one asset update")
	}

	for _, update := range p.Updates {
		// Validate denom
		if strings.TrimSpace(update.Denom) == "" {
			return errorsmod.Wrap(v1beta1.ErrInvalidLengthQuery, "asset denom cannot be empty")
		}

		// Validate magnitude
		validMagnitudes := map[string]bool{
			"small":  true,
			"medium": true,
			"high":   true,
		}
		if !validMagnitudes[update.Magnitude] {
			return errorsmod.Wrapf(v1beta1.ErrInvalidLengthQuery, "invalid magnitude: %s, must be one of small, medium, or high", update.Magnitude)
		}

		// Validate direction
		if update.Direction != "up" && update.Direction != "down" {
			return errorsmod.Wrapf(v1beta1.ErrInvalidLengthQuery, "invalid direction: %s, must be 'up' or 'down'", update.Direction)
		}
	}

	return nil
}
