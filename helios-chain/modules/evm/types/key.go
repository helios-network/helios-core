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
package types

import (
	"encoding/binary"

	"github.com/ethereum/go-ethereum/common"
)

const (
	// ModuleName string name of module
	ModuleName = "evm"

	// StoreKey key for ethereum storage data, account code (StateDB) or block
	// related data for Web3.
	// The EVM module should use a prefix store.
	StoreKey = ModuleName

	// ObjectStoreKey is the key to access the EVM object store, that is reset
	// during the Commit phase.
	ObjectStoreKey = "object:" + ModuleName

	// RouterKey uses module name for routing
	RouterKey = ModuleName
)

// prefix bytes for the EVM persistent store
const (
	prefixCode = iota + 1
	prefixStorage
	prefixParams
)

// prefix bytes for the EVM object store
const (
	prefixObjectBloom = iota + 1
	prefixObjectGasUsed
	prefixObjectParams
)

// KVStore key prefixes
var (
	KeyPrefixCode    = []byte{prefixCode}
	KeyPrefixStorage = []byte{prefixStorage}
	KeyPrefixParams  = []byte{prefixParams}
)

// Object Store key prefixes
var (
	KeyPrefixObjectBloom   = []byte{prefixObjectBloom}
	KeyPrefixObjectGasUsed = []byte{prefixObjectGasUsed}
	// cache the `EVMBlockConfig` during the whole block execution
	KeyPrefixObjectParams = []byte{prefixObjectParams}
)

// AddressStoragePrefix returns a prefix to iterate over a given account storage.
func AddressStoragePrefix(address common.Address) []byte {
	return append(KeyPrefixStorage, address.Bytes()...)
}

// StateKey defines the full key under which an account state is stored.
func StateKey(address common.Address, key []byte) []byte {
	return append(AddressStoragePrefix(address), key...)
}

func ObjectGasUsedKey(txIndex int) []byte {
	var key [1 + 8]byte
	key[0] = prefixObjectGasUsed
	binary.BigEndian.PutUint64(key[1:], uint64(txIndex))
	return key[:]
}

func ObjectBloomKey(txIndex, msgIndex int) []byte {
	var key [1 + 8 + 8]byte
	key[0] = prefixObjectBloom
	binary.BigEndian.PutUint64(key[1:], uint64(txIndex))
	binary.BigEndian.PutUint64(key[9:], uint64(msgIndex))
	return key[:]
}
