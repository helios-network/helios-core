// Copyright 2021 Evmos Foundation
// This file is part of Evmos' Ethermint library.
//
// The Ethermint library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The Ethermint library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the Ethermint library. 
package statedb

import (
	"math/big"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"
	evmtypes "helios-core/helios-chain/modules/evm/types"
)

// Keeper provide underlying storage of StateDB
type Keeper interface {
	GetParams(sdk.Context) evmtypes.Params

	Transfer(ctx sdk.Context, sender, recipient sdk.AccAddress, coins sdk.Coins) error
	AddBalance(ctx sdk.Context, addr sdk.AccAddress, coins sdk.Coins) error
	SubBalance(ctx sdk.Context, addr sdk.AccAddress, coins sdk.Coins) error
	SetBalance(ctx sdk.Context, addr common.Address, amount *big.Int, denom string) error
	GetBalance(ctx sdk.Context, addr sdk.AccAddress, denom string) *big.Int

	// Read methods
	GetAccount(ctx sdk.Context, addr common.Address) *Account
	GetState(ctx sdk.Context, addr common.Address, key common.Hash) common.Hash
	GetCode(ctx sdk.Context, codeHash common.Hash) []byte
	// the callback returns false to break early
	ForEachStorage(ctx sdk.Context, addr common.Address, cb func(key, value common.Hash) bool)

	// Write methods, only called by `StateDB.Commit()`
	SetAccount(ctx sdk.Context, addr common.Address, account Account) error
	SetState(ctx sdk.Context, addr common.Address, key common.Hash, value []byte)
	SetCode(ctx sdk.Context, codeHash []byte, code []byte)
	DeleteAccount(ctx sdk.Context, addr common.Address) error
}
