package types

import (
	"math/big"
	"testing"

	sdkmath "cosmossdk.io/math"
	"github.com/stretchr/testify/require"
)

func TestMaxInt256(t *testing.T) {
	maxInt256Plus1 := new(big.Int).Add(MaxInt256, big.NewInt(1))
	require.Equal(t, sdkmath.MaxBitLen, MaxInt256.BitLen())
	require.Equal(t, sdkmath.MaxBitLen+1, maxInt256Plus1.BitLen())

	require.True(t, IsValidInt256(MaxInt256))
	require.False(t, IsValidInt256(maxInt256Plus1))
}
