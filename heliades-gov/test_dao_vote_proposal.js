import { Wallet } from '@ethersproject/wallet'
import { createTxMsgVote } from '@helios-chain-labs/transactions'
import { broadcast, getSender, signTransaction } from '@helios-chain-labs/helios-ts-wallet'
import { createMsgVote } from '@helios-chain-labs/proto'
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
      'river census abuse hover mention bench tumble youth endorse silent machine ceiling burst village sock nuclear text tent argue barrel core entire beyond dream'

    const wallet = Wallet.fromMnemonic(privateMnemonic)
    console.log('Wallet Address:', wallet.address)
    console.log('Wallet Key:', wallet.privateKey)

    const sender = await getSender(wallet)

    // Define the vote parameters
    const proposalId = '1' // Replace with actual proposal ID you want to vote on
    const voteOption = protopkg.cosmosbetagov.gov.v1beta1.VoteOption.VOTE_OPTION_YES // Or other options like NO, ABSTAIN, NO_WITH_VETO

    // Create the vote transaction
    const txVote = createTxMsgVote(
      LOCALNET_CHAIN,
      sender,
      LOCALNET_FEE,
      '', // Optional memo
      {
        proposalId: proposalId,
        voter: sender.accountAddress,
        option: voteOption
      }
    )

    console.log('Vote Transaction:', txVote)

    // Sign the transaction
    const signedTx = await signTransaction(
      wallet,
      txVote,
      "BROADCAST_MODE_SYNC"
    )
    console.log('Signed Transaction:', signedTx)

    // Broadcast the transaction
    const broadcastRes = await broadcast(signedTx)
    console.log('Broadcast Response:', broadcastRes)

    if (broadcastRes.tx_response.code === 0) {
      console.log('Vote Transaction Success:', broadcastRes.tx_response.txhash)
    } else {
      console.error('Vote Transaction Failed:', broadcastRes.tx_response.raw_log)
    }
  } catch (error) {
    console.error('Error:', error)
  }
})()