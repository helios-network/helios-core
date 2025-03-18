package stream

import (
	"fmt"
	"helios-core/helios-chain/stream/types"
)

var ErrInvalidParameters = fmt.Errorf("firstMap and secondMap must have the same length")

func Filter[V types.BankBalance](itemMap map[string][]*V, filter []string) (out []*V) {
	wildcard := false
	if len(filter) > 0 {
		wildcard = filter[0] == "*"
	}
	if wildcard {
		for _, items := range itemMap {
			out = append(out, items...)
		}
		return
	}
	for _, marketID := range filter {
		if updates, ok := itemMap[marketID]; ok {
			out = append(out, updates...)
		}
	}
	return
}

func getMemAddr(i interface{}) string {
	return fmt.Sprintf("%p", i)
}
