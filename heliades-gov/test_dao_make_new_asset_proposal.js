import { Wallet } from '@ethersproject/wallet'
import { createTxMsgSubmitProposal } from '@helios-chain-labs/transactions'
import { broadcast, getSender, signTransaction } from '@helios-chain-labs/helios-ts-wallet'
// Import Protobuf package
import {createMsgSubmitProposal } from '@helios-chain-labs/proto'
import protopkg from '@helios-chain-labs/proto';

;(async () => {
  try {
    const LOCALNET_FEE = {
      amount: '20000000000000000',
      denom: 'ahelios',
      gas: '500000',
      feePayer: '',
      feeGranter: '',
    }

    const LOCALNET_CHAIN = {
      chainId: 42000,
      cosmosChainId: '42000',
      rpcEndpoint: 'http://localhost:26657',
      bech32Prefix: 'helios',
      currency: {
        coinDenom: 'HELIOS',
        coinMinimalDenom: 'ahelios',
        coinDecimals: 18,
      },
    }

    const privateMnemonic =
      'west mouse extra original dizzy dinosaur corn lottery access off slab surge piano build rabbit educate amused trophy orbit cable relax chimney trend inner'

    const wallet = Wallet.fromMnemonic(privateMnemonic)
    console.log('Wallet Address:', wallet.address)

    const sender = await getSender(wallet)

    const newAssetProposal = new protopkg.helios.erc20.v1.AddNewAssetConsensusProposal({
      title: 'Whitelist USDT into the consensus with a base stake of power 100',
      description: 'Explaining why USDT would be a good potential for Helios consensus and why it would secure the market',
      assets: [
        new protopkg.helios.erc20.v1.Asset({
          denom: 'USDT',
          contract_address: '0x1234567890abcdef1234567890abcdef12345678',
          chain_id: 'ethereum',
          decimals: 6,
          base_weight: 100,
          metadata: 'Tether stablecoin'
        }),
      ]
    });

    const newAssetProposalAny = new protopkg.google.protobuf.Any({
      type_url: '/helios.erc20.v1.AddNewAssetConsensusProposal',
      value: newAssetProposal.serializeBinary(),
    });

    // Create the governance proposal transaction
    const txProposal = createTxMsgSubmitProposal(
      LOCALNET_CHAIN,
      sender,
      LOCALNET_FEE,
      '', // Optional memo
      {
        content: newAssetProposalAny,
        initialDepositDenom: "ahelios",
        initialDepositAmount: "1000000000000000000",
        proposer: sender.accountAddress
      }
    )

    console.log('Proposal Transaction:', txProposal)

    // Sign the transaction
    const signedTx = await signTransaction(
      wallet,
      txProposal,
      "BROADCAST_MODE_SYNC"
    )
    console.log('Signed Transaction:', signedTx)

    // Broadcast the transaction
    const broadcastRes = await broadcast(signedTx)
    console.log('Broadcast Response:', broadcastRes)

    if (broadcastRes.tx_response.code === 0) {
      console.log('Transaction Success:', broadcastRes.tx_response.txhash)
    } else {
      console.error('Transaction Failed:', broadcastRes.tx_response.raw_log)
    }
  } catch (error) {
    console.error('Error:', error)
  }
})()
