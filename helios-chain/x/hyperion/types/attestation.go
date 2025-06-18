package types

import "strings"

func (a *Attestation) ContainsVote(address string) bool {
	prefix := address + ":"
	for _, vote := range a.Votes {
		if strings.HasPrefix(vote, prefix) {
			return true
		}
	}
	return false
}
