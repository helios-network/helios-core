package keeper

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"image"
	"strconv"

	"cosmossdk.io/log"

	"cosmossdk.io/store/prefix"
	storetypes "cosmossdk.io/store/types"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"helios-core/helios-chain/x/logos/types"
)

type Keeper struct {
	cdc       codec.BinaryCodec
	storeKey  storetypes.StoreKey
	authority sdk.AccAddress
}

func NewKeeper(
	cdc codec.BinaryCodec,
	storeKey storetypes.StoreKey,
	authority sdk.AccAddress,
) *Keeper {

	return &Keeper{
		cdc:       cdc,
		storeKey:  storeKey,
		authority: authority,
	}
}

func (k *Keeper) Logger(ctx sdk.Context) log.Logger {
	return ctx.Logger().With("module", types.ModuleName)
}

// SetLogo stocke un logo dans le store
func (k Keeper) SetLogo(ctx sdk.Context, logo types.Logo) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.LogoKey)

	b := k.cdc.MustMarshal(&logo)
	store.Set([]byte(logo.Hash), b)
}

// GetLogo récupère un logo par son hash
func (k Keeper) GetLogo(ctx sdk.Context, hash string) (types.Logo, bool) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.LogoKey)
	b := store.Get([]byte(hash))
	if b == nil {
		return types.Logo{}, false
	}

	var logo types.Logo
	k.cdc.MustUnmarshal(b, &logo)
	return logo, true
}

func (k Keeper) ValidateBase64Logo(ctx sdk.Context, data string) error {
	params := k.GetParams(ctx)

	if len(data) > (int(params.MaxLogoSize) * 4 / 3) { // Base64 emplify the size by ~33%
		return fmt.Errorf("logo size exceeds maximum allowed size of %d bytes", params.MaxLogoSize)
	}

	// 2. Decoding the base64
	dataBytes, err := base64.StdEncoding.DecodeString(data)
	if err != nil {
		return errors.New("logo is not valid base64")
	}

	// 3. Check PNG format
	img, format, err := image.Decode(bytes.NewReader(dataBytes))
	if err != nil || format != "png" {
		return errors.New("logo must be a valid PNG image")
	}

	// 4. Check the picture size
	bounds := img.Bounds()
	width, height := bounds.Dx(), bounds.Dy()
	if width != 200 || height != 200 {
		return errors.New("logo must be 200x200 pixels")
	}

	return nil
}

// StoreLogo stores a new logo and generates its hash
func (k Keeper) StoreLogo(ctx sdk.Context, data string) (string, error) {
	if err := k.ValidateBase64Logo(ctx, data); err != nil {
		return "", err
	}

	// Generate a SHA-256 hash of the content
	hasher := sha256.New()
	hasher.Write([]byte(data))
	hash := hex.EncodeToString(hasher.Sum(nil))

	// Create the logo object
	logo := types.Logo{
		Hash:      hash,
		Data:      data,
		CreatedAt: strconv.FormatInt(ctx.BlockHeight(), 10),
	}

	// Store the logo
	k.SetLogo(ctx, logo)

	return hash, nil
}

// GetAllLogos retrieves all logos
func (k Keeper) GetAllLogos(ctx sdk.Context) []types.Logo {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.LogoKey)
	iterator := store.Iterator(nil, nil)
	defer iterator.Close()

	var logos []types.Logo
	for ; iterator.Valid(); iterator.Next() {
		var logo types.Logo
		k.cdc.MustUnmarshal(iterator.Value(), &logo)
		logos = append(logos, logo)
	}

	return logos
}

// GetParams retrieves the module parameters
func (k Keeper) GetParams(ctx sdk.Context) types.Params {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.ParamsKey)

	var params types.Params
	k.cdc.MustUnmarshal(store.Get(sdk.Uint64ToBigEndian(0)), &params)
	return params
}

// SetParams sets the module parameters
func (k Keeper) SetParams(ctx sdk.Context, params types.Params) error {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.ParamsKey)

	bz, err := k.cdc.Marshal(&params)
	if err != nil {
		return err
	}
	store.Set(sdk.Uint64ToBigEndian(0), bz)

	return nil
}
