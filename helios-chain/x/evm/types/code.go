package types

import (
	"bytes"

	"github.com/ethereum/go-ethereum/crypto"
)

func FunctionExists(code []byte, signature string) bool {
	hash := crypto.Keccak256([]byte(signature))
	selector := hash[:4]
	return bytes.Contains(code, selector)
}
