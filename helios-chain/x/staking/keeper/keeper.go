package keeper

import (
	addresscodec "cosmossdk.io/core/address"
	storetypes "cosmossdk.io/core/store"
	"github.com/cosmos/cosmos-sdk/codec"
	slashingkeeper "github.com/cosmos/cosmos-sdk/x/slashing/keeper"
	stakingkeeper "github.com/cosmos/cosmos-sdk/x/staking/keeper"
	"github.com/cosmos/cosmos-sdk/x/staking/types"
)

// Keeper is a wrapper around the Cosmos SDK staking keeper.
type Keeper struct {
	*stakingkeeper.Keeper // Embedded value, not pointer
	ak                    types.AccountKeeper
	bk                    types.BankKeeper
}

// NewKeeper creates a new staking Keeper wrapper instance.
func NewKeeper(
	cdc codec.BinaryCodec,
	storeService storetypes.KVStoreService,
	ak types.AccountKeeper,
	bk types.BankKeeper,
	ec types.Erc20Keeper,
	authority string,
	validatorAddressCodec addresscodec.Codec,
	consensusAddressCodec addresscodec.Codec,
) *Keeper {
	return &Keeper{
		Keeper: stakingkeeper.NewKeeper(cdc, storeService, ak, bk, ec, authority, validatorAddressCodec, consensusAddressCodec), // Dereference the pointer
		ak:     ak,
		bk:     bk,
	}
}

func (k *Keeper) SetErc20Keeper(erc20Keeper types.Erc20Keeper) {
	k.Keeper.SetErc20Keeper(erc20Keeper)
}

func (k *Keeper) SetSlashingKeeper(slk *slashingkeeper.Keeper) {
	k.Keeper.SetSlashingKeeper(slk)
}
