import { Wallet } from '@ethersproject/wallet'
import { createTxMsgSubmitProposal } from '@helios-chain-labs/transactions'
import { broadcast, getSender, signTransaction } from '@helios-chain-labs/helios-ts-wallet'
import protopkg from '@helios-chain-labs/proto'

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
      chainId: 4242,
      cosmosChainId: '4242',
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

    // First, get the current inflation rate to display in logs
    // try {
    //   const response = await fetch(`${LOCALNET_CHAIN.rpcEndpoint.replace('26657', '1317')}/cosmos/mint/v1beta1/inflation`);
    //   const data = await response.json();
    //   console.log('Current Inflation Rate:', data.inflation);
    // } catch (error) {
    //   console.log('Could not fetch current inflation rate:', error.message);
    // }

    // Create an UpdateInflationProposal to increase early phase rate by 1%
    const updateInflationProposal = new protopkg.cosmosmint.mint.v1beta1.UpdateInflationProposal({
      title: 'Increase Early Phase Inflation Rate by 1%',
      description: 'This proposal increases the early phase inflation rate by 1% to stimulate network growth and validator participation.',
      phase: 'early',
      // Based on the error message, we need to match the scale of 0.150000000000000000
      // For 16%, we need 0.160000000000000000
      // Which as an integer would be 160000000000000000
      new_rate: '300000000000000000' // 16% as an integer with the correct scale
    });


    const updateInflationProposalAny = new protopkg.google.protobuf.Any({
      // Use the correct type URL matching the codec registration
      type_url: '/cosmos.mint.v1beta1.UpdateInflationProposal',
      value: updateInflationProposal.serializeBinary(),
    });

    // Create the governance proposal transaction
    const txProposal = createTxMsgSubmitProposal(
      LOCALNET_CHAIN,
      sender,
      LOCALNET_FEE,
      'Testing inflation rate update', // Optional memo
      {
        content: updateInflationProposalAny,
        initialDepositDenom: "ahelios",
        initialDepositAmount: "10000000000000000000", // 10 HELIOS to ensure it passes deposit threshold
        proposer: sender.accountAddress
      }
    )

    console.log('Proposal Transaction Created')

    // Sign the transaction
    const signedTx = await signTransaction(
      wallet,
      txProposal,
      "BROADCAST_MODE_SYNC"
    )
    console.log('Transaction Signed')

    // Broadcast the transaction
    const broadcastRes = await broadcast(signedTx)
    console.log('Broadcast Response:', broadcastRes)

    if (broadcastRes.tx_response.code === 0) {
      console.log('Transaction Success! Proposal submitted with hash:', broadcastRes.tx_response.txhash)

      //console.log(broadcastRes.tx_response)
    } else {
      console.error('Transaction Failed:', broadcastRes.tx_response.raw_log)
    }
  } catch (error) {
    console.error('Error:', error)
  }
})()