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
	"bytes"
	"sort"

	"github.com/ethereum/go-ethereum/common"
	"helios-core/helios-chain/modules/evm/types"
)

// Account is the Ethereum consensus representation of accounts.
// These objects are stored in the storage of auth module.
type Account struct {
	Nonce    uint64
	CodeHash []byte
}

// NewEmptyAccount returns an empty account.
func NewEmptyAccount() *Account {
	return &Account{
		CodeHash: types.EmptyCodeHash,
	}
}

// IsContract returns if the account contains contract code.
func (acct Account) IsContract() bool {
	return !bytes.Equal(acct.CodeHash, types.EmptyCodeHash)
}

// Storage represents in-memory cache/buffer of contract storage.
type Storage map[common.Hash]common.Hash

// SortedKeys sort the keys for deterministic iteration
func (s Storage) SortedKeys() []common.Hash {
	keys := make([]common.Hash, len(s))
	i := 0
	for k := range s {
		keys[i] = k
		i++
	}
	sort.Slice(keys, func(i, j int) bool {
		return bytes.Compare(keys[i].Bytes(), keys[j].Bytes()) < 0
	})
	return keys
}

// stateObject is the state of an acount
type stateObject struct {
	db *StateDB

	originalAccount *Account // Account original data without any change applied, nil means it was not existent
	account         Account  // Account data with all mutations applied in the scope of block

	code []byte

	// state storage
	originStorage Storage // Storage entries that have been accessed within the current block
	dirtyStorage  Storage // Storage entries that have been modified within the current transaction

	// overridden state, when not nil, replace the whole committed state,
	// mainly to support the stateOverrides in eth_call.
	overrideStorage Storage

	// address of ethereum account
	address common.Address

	// Flag whether the account was marked as self-destructed. The self-destructed
	// account is still accessible in the scope of same transaction.
	selfDestructed bool

	// This is an EIP-6780 flag indicating whether the object is eligible for
	// self-destruct according to EIP-6780. The flag could be set either when
	// the contract is just created within the current transaction, or when the
	// object was previously existent and is being deployed as a contract within
	// the current transaction.
	newContract bool
}

// newObject creates a state object
func newObject(db *StateDB, address common.Address, acct *Account) *stateObject {
	origin := acct
	if acct == nil {
		acct = NewEmptyAccount()
	}
	return &stateObject{
		db:              db,
		address:         address,
		originalAccount: origin,
		account:         *acct,
		originStorage:   make(Storage),
		dirtyStorage:    make(Storage),
	}
}

// codeDirty returns whether the codeHash is modified
func (s *stateObject) codeDirty() bool {
	return s.originalAccount == nil || !bytes.Equal(s.account.CodeHash, s.originalAccount.CodeHash)
}

// nonceDirty returns whether the nonce is modified
func (s *stateObject) nonceDirty() bool {
	return s.originalAccount == nil || s.account.Nonce != s.originalAccount.Nonce
}

// empty returns whether the account is considered empty.
func (s *stateObject) empty() bool {
	return s.account.Nonce == 0 && bytes.Equal(s.account.CodeHash, types.EmptyCodeHash)
}

func (s *stateObject) markSelfDestructed() {
	s.selfDestructed = true
}

//
// Attribute accessors
//

// Returns the address of the contract/account
func (s *stateObject) Address() common.Address {
	return s.address
}

// Code returns the contract code associated with this object, if any.
func (s *stateObject) Code() []byte {
	if s.code != nil {
		return s.code
	}
	if bytes.Equal(s.CodeHash(), types.EmptyCodeHash) {
		return nil
	}
	code := s.db.keeper.GetCode(s.db.ctx, common.BytesToHash(s.CodeHash()))
	s.code = code
	return code
}

// CodeSize returns the size of the contract code associated with this object,
// or zero if none.
func (s *stateObject) CodeSize() int {
	return len(s.Code())
}

// SetCode set contract code to account
func (s *stateObject) SetCode(codeHash common.Hash, code []byte) {
	prevcode := s.Code()
	s.db.journal.append(codeChange{
		account:  &s.address,
		prevhash: s.CodeHash(),
		prevcode: prevcode,
	})
	s.setCode(codeHash, code)
}

func (s *stateObject) setCode(codeHash common.Hash, code []byte) {
	s.code = code
	s.account.CodeHash = codeHash[:]
}

// SetCode set nonce to account
func (s *stateObject) SetNonce(nonce uint64) {
	s.db.journal.append(nonceChange{
		account: &s.address,
		prev:    s.account.Nonce,
	})
	s.setNonce(nonce)
}

func (s *stateObject) setNonce(nonce uint64) {
	s.account.Nonce = nonce
}

// CodeHash returns the code hash of account
func (s *stateObject) CodeHash() []byte {
	return s.account.CodeHash
}

// Nonce returns the nonce of account
func (s *stateObject) Nonce() uint64 {
	return s.account.Nonce
}

// GetCommittedState query the committed state
func (s *stateObject) GetCommittedState(key common.Hash) common.Hash {
	if s.overrideStorage != nil {
		if value, ok := s.overrideStorage[key]; ok {
			return value
		}
		return common.Hash{}
	}

	if value, cached := s.originStorage[key]; cached {
		return value
	}
	// If no live objects are available, load it from keeper
	value := s.db.keeper.GetState(s.db.ctx, s.Address(), key)
	s.originStorage[key] = value
	return value
}

// GetState query the current state (including dirty state)
func (s *stateObject) GetState(key common.Hash) common.Hash {
	if value, dirty := s.dirtyStorage[key]; dirty {
		return value
	}
	return s.GetCommittedState(key)
}

// SetState sets the contract state
func (s *stateObject) SetState(key common.Hash, value common.Hash) {
	// If the new value is the same as old, don't set
	prev := s.GetState(key)
	if prev == value {
		return
	}
	// New value is different, update and journal the change
	s.db.journal.append(storageChange{
		account:  &s.address,
		key:      key,
		prevalue: prev,
	})
	s.setState(key, value)
}

func (s *stateObject) SetStorage(storage Storage) {
	s.overrideStorage = storage
	s.originStorage = make(Storage)
	s.dirtyStorage = make(Storage)
}

func (s *stateObject) setState(key, value common.Hash) {
	s.dirtyStorage[key] = value
}
