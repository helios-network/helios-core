import { Wallet } from '@ethersproject/wallet'
import { broadcast, getSender, signTransaction } from '@helios-chain-labs/helios-ts-wallet'
import { createTxMsgScheduleEVMCall } from '@helios-chain-labs/transactions'

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

    // Sample ERC20 ABI for transfer method
    const ERC20_ABI_JSON = JSON.stringify([
      {
        "constant": false,
        "inputs": [
          {
            "name": "_to",
            "type": "address"
          },
          {
            "name": "_value",
            "type": "uint256"
          }
        ],
        "name": "transfer",
        "outputs": [
          {
            "name": "",
            "type": "bool"
          }
        ],
        "payable": false,
        "stateMutability": "nonpayable",
        "type": "function"
      }
    ]);

    // Replace with your actual ERC20 token contract address
    const contractAddress = "0x1234567890123456789012345678901234567890";
    const recipientAddress = "0x0987654321098765432109876543210987654321";
    const transferAmount = "1000000000000000000"; // 1 token with 18 decimals

    // Use our new function to create the transaction
    const txSchedule = createTxMsgScheduleEVMCall(
      LOCALNET_CHAIN,
      sender,
      LOCALNET_FEE,
      'Testing Chronos EVM scheduler',
      {
        ownerAddress: sender.accountAddress,
        contractAddress: contractAddress,
        abiJson: ERC20_ABI_JSON,
        methodName: "transfer",
        params: [recipientAddress, transferAmount],
        frequency: 1,
        expirationBlock: 0,
        gasLimit: 200000
      }
    );

    console.log('Schedule Transaction Created');

    // Sign the transaction
    const signedTx = await signTransaction(
      wallet,
      txSchedule,
      "BROADCAST_MODE_SYNC"
    );
    console.log('Transaction Signed');

    // Broadcast the transaction
    const broadcastRes = await broadcast(signedTx);
    console.log('Broadcast Response:', broadcastRes);
    
    if (broadcastRes.tx_response.code === 0) {
      console.log('Transaction Success! Schedule submitted with hash:', broadcastRes.tx_response.txhash);
    } else {
      console.error('Transaction Failed:', broadcastRes.tx_response.raw_log);
    }
  } catch (error) {
    console.error('Error:', error);
  }
})();