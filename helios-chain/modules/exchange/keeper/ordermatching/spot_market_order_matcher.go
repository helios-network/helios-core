package ordermatching

import (
	"cosmossdk.io/math"

	"helios-core/helios-chain/modules/exchange/types"
)

type SpotMarketOrderbook struct {
	notional      math.LegacyDec
	totalQuantity math.LegacyDec

	orders         []*types.SpotMarketOrder
	fillQuantities []math.LegacyDec
	orderIdx       int
}

func NewSpotMarketOrderbook(
	spotMarketOrders []*types.SpotMarketOrder,
) *SpotMarketOrderbook {
	if len(spotMarketOrders) == 0 {
		return nil
	}

	fillQuantities := make([]math.LegacyDec, len(spotMarketOrders))
	for idx := range spotMarketOrders {
		fillQuantities[idx] = math.LegacyZeroDec()
	}

	orderGroup := SpotMarketOrderbook{
		notional:      math.LegacyZeroDec(),
		totalQuantity: math.LegacyZeroDec(),

		orders:         spotMarketOrders,
		fillQuantities: fillQuantities,
		orderIdx:       0,
	}

	return &orderGroup
}

func (b *SpotMarketOrderbook) GetNotional() math.LegacyDec                  { return b.notional }
func (b *SpotMarketOrderbook) GetTotalQuantityFilled() math.LegacyDec       { return b.totalQuantity }
func (b *SpotMarketOrderbook) GetOrderbookFillQuantities() []math.LegacyDec { return b.fillQuantities }
func (b *SpotMarketOrderbook) Done() bool                                   { return b.orderIdx == len(b.orders) }
func (b *SpotMarketOrderbook) Peek() *types.PriceLevel {
	if b.Done() {
		return nil
	}

	if b.fillQuantities[b.orderIdx].Equal(b.orders[b.orderIdx].OrderInfo.Quantity) {
		b.orderIdx++
		return b.Peek()
	}

	return &types.PriceLevel{
		Price:    b.orders[b.orderIdx].OrderInfo.Price,
		Quantity: b.orders[b.orderIdx].OrderInfo.Quantity.Sub(b.fillQuantities[b.orderIdx]),
	}
}

func (b *SpotMarketOrderbook) Fill(fillQuantity math.LegacyDec) error {
	newFillAmount := b.fillQuantities[b.orderIdx].Add(fillQuantity)

	if newFillAmount.GT(b.orders[b.orderIdx].OrderInfo.Quantity) {
		return types.ErrOrderbookFillInvalid
	}

	b.fillQuantities[b.orderIdx] = newFillAmount
	b.notional = b.notional.Add(fillQuantity.Mul(b.orders[b.orderIdx].OrderInfo.Price))
	b.totalQuantity = b.totalQuantity.Add(fillQuantity)

	return nil
}
