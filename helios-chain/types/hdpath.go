// Copyright Tharsis Labs Ltd.(Evmos)
// SPDX-License-Identifier:ENCL-1.0(https://github.com/evmos/evmos/blob/main/LICENSE)
package types

import (
	ethaccounts "github.com/ethereum/go-ethereum/accounts"
)

type (
	HDPathIterator func() ethaccounts.DerivationPath
)

// HDPathIterator receives a base path as a string and a boolean for the desired iterator type and
// returns a function that iterates over the base HD path, returning the string.
func NewHDPathIterator(basePath string, ledgerIter bool) (HDPathIterator, error) {
	hdPath, err := ethaccounts.ParseDerivationPath(basePath)
	if err != nil {
		return nil, err
	}

	if ledgerIter {
		return ethaccounts.LedgerLiveIterator(hdPath), nil
	}

	return ethaccounts.DefaultIterator(hdPath), nil
}
