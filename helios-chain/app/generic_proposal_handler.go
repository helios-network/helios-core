package app

import (
	"fmt"
	"reflect"

	baseapp "github.com/cosmos/cosmos-sdk/baseapp"
	codec "github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	govkeeper "github.com/cosmos/cosmos-sdk/x/gov/keeper"
	govv1beta1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1beta1"
)

func NewGenericProposalHandler(
	appCodec codec.Codec,
	msr *baseapp.MsgServiceRouter,
	govKeeper govkeeper.Keeper,
) govv1beta1.Handler {
	return func(ctx sdk.Context, c govv1beta1.Content) error {
		p, ok := c.(*govv1beta1.ModuleExecProposal)
		if !ok {
			return fmt.Errorf("unexpected content type %T", c)
		}

		if err := p.ValidateBasic(); err != nil {
			return err
		}

		for _, a := range p.Messages {
			var msg sdk.Msg
			if err := appCodec.UnpackAny(a, &msg); err != nil {
				return fmt.Errorf("failed to unpack msg: %w", err)
			}

			overrideAuthority(msg, govKeeper.GetAuthority())

			h := msr.Handler(msg)
			if h == nil {
				return fmt.Errorf("no handler found for message type %T", msg)
			}

			if _, err := h(ctx, msg); err != nil {
				return fmt.Errorf("failed to execute %T: %w", msg, err)
			}
		}

		return nil
	}
}

func overrideAuthority(msg sdk.Msg, newAuthority string) {
	v := reflect.ValueOf(msg)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	if v.Kind() != reflect.Struct {
		return
	}

	field := v.FieldByName("Authority")
	if field.IsValid() && field.CanSet() && field.Kind() == reflect.String {
		oldValue := field.String()
		if oldValue != newAuthority {
			field.SetString(newAuthority)
			fmt.Printf("→ Authority overridden (%T): %q → %q\n", msg, oldValue, newAuthority)
		}
	}

	signerField := v.FieldByName("Signer")
	if signerField.IsValid() && signerField.CanSet() && signerField.Kind() == reflect.String {
		oldValue := signerField.String()
		if oldValue != newAuthority {
			signerField.SetString(newAuthority)
			fmt.Printf("→ Authority overridden (%T): %q → %q\n", msg, oldValue, newAuthority)
		}
	}
}
