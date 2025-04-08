# Extended Transfers

On sendToHelios method the field data can contains an transaction extension.

```sol
function sendToHelios(
        address _tokenContract,
        bytes32 _destination,
        uint256 _amount,
        string calldata _data
    )
```

this extension can be used for execute other transaction once the transfer was complete.

`data` can contains json structured Messages:

```json
{
    "@type":"/helios.hyperion.v1.MsgSendToChain" ,
    "sender" :"helios1zun8av07cvqcfr2t29qwnh8u",
    "dest_hyperion_id": 21,
    "dest": "0x17267eB1FEC301848d4B5140eDDCFC48945427Ab",
    "amount": {"denom": "ahelios", "amount":"10"},
    "bridge_fee": {"denom":"ahelios", "amount":"10"}
}
```

in the hyperion module keeper/attestation.go we handle the .data messages once the transfer was completed.

exemple how to build json messages in go:

```go
msg := types.MsgSendToChain{
    Sender:         sdk.AccAddress(common.FromHex(claim.EthereumSender)).String(),
    DestHyperionId: claim.HyperionId,
    Dest:           claim.EthereumSender,
    Amount:         sdk.NewCoin("ahelios", math.NewInt(10)),
    BridgeFee:      sdk.NewCoin("ahelios", math.NewInt(10)),
}

type SignedMsg struct {
    AtType         string   `json:"@type"`

    // your fields
    Sender         string   `protobuf:"bytes,1,opt,name=sender,proto3" json:"sender,omitempty"`
    DestHyperionId uint64   `protobuf:"varint,2,opt,name=dest_hyperion_id,json=destHyperionId,proto3" json:"dest_hyperion_id,omitempty"`
    Dest           string   `protobuf:"bytes,3,opt,name=dest,proto3" json:"dest,omitempty"`
    Amount         sdk.Coin `protobuf:"bytes,4,opt,name=amount,proto3" json:"amount"`
    BridgeFee      sdk.Coin `protobuf:"bytes,5,opt,name=bridge_fee,json=bridgeFee,proto3" json:"bridge_fee"`
}

signedMessage := SignedMsg{
    AtType:         "/helios.hyperion.v1.MsgSendToChain", // message type
    Sender:         msg.Sender,
    DestHyperionId: msg.DestHyperionId,
    Dest:           msg.Dest,
    Amount:         msg.Amount,
    BridgeFee:      msg.BridgeFee,
}
signedMsgJSON, err := json.Marshal(signedMessage)
signedMsgJSONString := string(signedMsgJSON)
```

how to deserialize the message into attestation.go:

```go
func (k *Keeper) ValidateClaimData(ctx sdk.Context, claimData string, ethereumSigner sdk.AccAddress) (*sdk.Msg, error) {
    var data types.ClaimData

    if err := json.Unmarshal([]byte(claimData), &data); err == nil {
        if data.Metadata != nil {
            claimData = data.Data
        }
    }
    var msg sdk.Msg

    // Check if the claim data is a valid sdk msg
    if err := k.cdc.UnmarshalInterfaceJSON([]byte(claimData), &msg); err != nil {
        return nil, nil
    }

    var message *types.MsgSendToChain

    message, ok := msg.(*types.MsgSendToChain)
    if !ok {
        return nil, errors.Errorf("claim data is not a valid MsgSendToChain")
    }

    // Enforce that msg.ValidateBasic() succeeds
    if err := message.ValidateBasic(); err != nil {
        return nil, errors.Errorf("claim data is not a valid sdk.Msg ValidateBasic: %s", "err", err.Error())
    }

    // Enforce that the claim data is signed by the ethereum signer
    if !message.GetSigners()[0].Equals(ethereumSigner) {
        return nil, errors.Errorf("claim data is not signed by ethereum signer: %s", ethereumSigner.String())
    }

    return &msg, nil
}
```
