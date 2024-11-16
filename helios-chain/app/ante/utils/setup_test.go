package utils_test

import (
	"testing"

	"github.com/stretchr/testify/suite"
	"helios-core/helios-chain/app/ante/testutils"
)

type AnteTestSuite struct {
	*testutils.AnteTestSuite
}

func TestAnteTestSuite(t *testing.T) {
	baseSuite := new(testutils.AnteTestSuite)
	baseSuite.WithLondonHardForkEnabled(true)

	suite.Run(t, &AnteTestSuite{
		AnteTestSuite: baseSuite,
	})
}
