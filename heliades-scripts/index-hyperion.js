const ethers = require('ethers');
const WebSocket = require('ws');
const fs = require('fs');

// const RPC_URL = 'https://testnet1.helioschainlabs.org';
const RPC_URL = 'http://localhost:8545';
const COSMOS_RPC_WS = 'ws://localhost:26657/websocket'; // WebSocket Cosmos RPC

const PRIVATE_KEY = '2c37c3d09d7a1c957f01ad200cec69bc287d0a9cc85b4dce694611a4c9c24036';

const PRIVATE_KEY2 = 'e1ab51c450698b0af4722e074e39394bd99822f0b00f1a787a131b48c14d4483'

const PRECOMPILE_CONTRACT_ADDRESS = '0x0000000000000000000000000000000000000806';

const provider = new ethers.JsonRpcProvider(RPC_URL);

const wallet = new ethers.Wallet(PRIVATE_KEY, provider);
const wallet2 = new ethers.Wallet(PRIVATE_KEY2, provider);

async function hyperionProposal({ title, description, msg }) {
  const abi = JSON.parse(fs.readFileSync('../helios-chain/precompiles/gov/abi.json').toString()).abi;
  const contract = new ethers.Contract("0x0000000000000000000000000000000000000805", abi, wallet);

  try {
    console.log('Ajout d\'une nouvelle proposition au consensus...');
    console.log('Arguments envoyés au contrat :', { title, description });
    
    const tx = await contract.hyperionProposal(title, description, msg, "1000000000000000000", {
      gasPrice: 50000000000,
      gasLimit: 5000000
    });
    console.log('Transaction envoyée, hash :', tx.hash);

    const receipt = await tx.wait();
    console.log('Transaction confirmée dans le bloc :', receipt.blockNumber);

    console.log('Proposition soumise avec succès !');
  } catch (error) {
    console.error('Erreur lors de la soumission de la proposition :', error);
  }
}

async function vote(proposalId){
  const abi = JSON.parse(fs.readFileSync('../helios-chain/precompiles/gov/abi.json').toString()).abi;
  const contract = new ethers.Contract("0x0000000000000000000000000000000000000805", abi, wallet);

  try {
    console.log('vote en cours...');
    
    const tx = await contract.vote(wallet.address, proposalId, 1, "voting testtest");
    console.log('Transaction envoyée, hash :', tx.hash);

    const receipt = await tx.wait();
    console.log('Transaction confirmée dans le bloc :', receipt.blockNumber);

    console.log('vote fait a success soumise avec succès !');
  } catch (error) {
    console.error('Erreur lors de la soumission de la proposition :', error);
  }
}

async function main() {
  await hyperionProposal({
    title: "test",
    description: "test",
    msg: JSON.stringify({
      "@type": "/helios.hyperion.v1.MsgAddCounterpartyChainParams",
      "counterparty_chain_params": {
        "hyperion_id": 1,
        "contract_source_hash": "", 
        "bridge_counterparty_address": "0xB8ed88AcD8b7ac80d9f546F4D75F33DD19dD5746",
        "bridge_chain_id": 1,
        "bridge_chain_name": "Test Chain",
        "bridge_chain_logo": "",
        "bridge_chain_type": "evm",
        "signed_valsets_window": 25000,
        "signed_batches_window": 25000,
        "signed_claims_window": 25000,
        "target_batch_timeout": 3600000,
        "average_block_time": 2000,
        "average_counterparty_block_time": 15000,
        "slash_fraction_valset": "0.001",
        "slash_fraction_batch": "0.001", 
        "slash_fraction_claim": "0.001",
        "slash_fraction_conflicting_claim": "0.001",
        "unbond_slashing_valsets_window": 25000,
        "slash_fraction_bad_eth_signature": "0.001",
        "bridge_contract_start_height": 0,
        "valset_reward": {
          "denom": "ahelios",
          "amount": "0"
        },
        "initializer": wallet.address,
        "default_tokens": [
          {
            "token_address_to_denom": {
              "denom": "ahelios",
              "token_address": "0x462D63407eb86531dce7f948F2145382bc269E7C",
              "is_cosmos_originated": true,
              "is_concensus_token": true,
              "symbol": "HLS",
              "decimals": 18
            },
            "default_holders": [
            ]
          }
        ],
        "rpcs": [],
        "offset_valset_nonce": 0,
        "min_call_external_data_gas": 10000000,
        "paused": true
      }
    })
  });

  // await vote(1);
}

main();