package types

import (
	"bytes"
	"embed"
	"encoding/json"
	"fmt"
)

// DefaultParamspace defines the default logos module parameter subspace
const (
	DefaultParamspace = ModuleName
)

// DefaultParams returns a copy of the default params
func DefaultParams() *Params {
	return &Params{
		MaxLogoSize: 1024 * 1024, // 1MB
	}
}

//go:embed default-logos/80002.json
//go:embed default-logos/HLS.json
//go:embed default-logos/ETH.json
//go:embed default-logos/BNB.json
//go:embed default-logos/AVAX.json
var defaultLogos embed.FS

func DefaultLogo() []*Logo {

	files := []string{"default-logos/80002.json", "default-logos/HLS.json", "default-logos/ETH.json", "default-logos/BNB.json", "default-logos/AVAX.json"}

	lstLogos := []*Logo{}

	for _, file := range files {
		data, err := defaultLogos.ReadFile(file)
		if err != nil {
			panic(err)
		}

		var msg Logo
		if err := json.Unmarshal(data, &msg); err == nil {
			lstLogos = append(lstLogos, &msg)
		}
	}
	return lstLogos
}

// ValidateBasic checks that the parameters have valid values.
func (p Params) ValidateBasic() error {
	if p.MaxLogoSize < 1024*1024 {
		return fmt.Errorf("max logo size must be at least 1MB")
	}
	return nil
}

// Equal returns a boolean determining if two Params types are identical.
func (p Params) Equal(p2 Params) bool {
	bz1 := ModuleCdc.MustMarshalLengthPrefixed(&p)
	bz2 := ModuleCdc.MustMarshalLengthPrefixed(&p2)
	return bytes.Equal(bz1, bz2)
}
