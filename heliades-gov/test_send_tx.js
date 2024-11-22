import { Wallet } from '@ethersproject/wallet'
import { createMessageSend, createTxMsgSubmitProposal } from '@helios-chain-labs/transactions'
import {
  broadcast,
  getSender,
  signTransactionUsingEIP712,
} from '@helios-chain-labs/helios-ts-wallet'

;(async () => {
  const LOCALNET_FEE = {
    amount: '20000000000000000',
    denom: 'ahelios',
    gas: '200000',
    feePayer: '', // Will be filled with sender address
    feeGranter: '' // Leave empty if not using fee granter
}
  
  const LOCALNET_CHAIN = {
    chainId: 4242,
    cosmosChainId: '4242',
    rpcEndpoint: 'http://localhost:26657',
    // Add these additional parameters
    bech32Prefix: 'helios',
    currency: {
        coinDenom: 'HELIOS',
        coinMinimalDenom: 'ahelios',
        coinDecimals: 18,
    }
}

  const privateMnemonic =
    'west mouse extra original dizzy dinosaur corn lottery access off slab surge piano build rabbit educate amused trophy orbit cable relax chimney trend inner'
  
  const wallet = Wallet.fromMnemonic(privateMnemonic)
  console.log(wallet.address)
  
  //return
  // Add error handling for getSender
  let sender;
  try {
    sender = await getSender(wallet)
  } catch (error) {
    console.error('Error getting sender:', error)
    return
  }

  // Detailed transaction creation
  const txSimple = createMessageSend(
    LOCALNET_CHAIN, 
    sender, 
    LOCALNET_FEE, 
    '', 
    {
      destinationAddress: 'helios1h4mxjgyjuqsfq42ut50tg2g6kz4fml24x4zk8s',
      amount: '1',
      denom: 'ahelios',
    }
  )


  // Enhanced signing with more explicit parameters
  const resMM = await signTransactionUsingEIP712(
    wallet,
    sender.accountAddress,
    txSimple,
    {
      chainId: 4242,
      cosmosChainId: 'helios_4242-1', // Use full chain ID
    },
    "BROADCAST_MODE_SYNC"
  )

  // Comprehensive error handling for broadcast
  try {
    const broadcastRes = await broadcast(resMM)
    console.log('Broadcast Response:', broadcastRes)
    
    if (broadcastRes.tx_response.code === 0) {
      console.log('Transaction Success')
      console.log('Transaction Hash:', broadcastRes.tx_response.txhash)
    } else {
      console.log('Transaction Failed')
      console.log('Error Code:', broadcastRes.tx_response.code)
      console.log('Error Message:', broadcastRes.tx_response.raw_log)
    }
  } catch (broadcastError) {
    console.error('Broadcast Error:', broadcastError)
  }
})()