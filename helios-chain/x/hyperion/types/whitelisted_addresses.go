package types

func (w *WhitelistedAddresses) ContainsAddress(address string) bool {
	for _, a := range w.Addresses {
		if a == address {
			return true
		}
	}
	return false
}
