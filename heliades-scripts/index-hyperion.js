const ethers = require('ethers');
const WebSocket = require('ws');
const fs = require('fs');

// const RPC_URL = 'https://testnet1.helioschainlabs.org';
const RPC_URL = 'http://localhost:8545';

const PRIVATE_KEY = '';

const provider = new ethers.JsonRpcProvider(RPC_URL);

const wallet = new ethers.Wallet(PRIVATE_KEY, provider);

async function hyperionProposal({ title, description, msg }) {
  const abi = JSON.parse(fs.readFileSync('../helios-chain/precompiles/gov/abi.json').toString()).abi;
  const contract = new ethers.Contract("0x0000000000000000000000000000000000000805", abi, wallet);

  try {
    console.log('Ajout d\'une nouvelle proposition au consensus...');
    console.log('Arguments envoyés au contrat :', { title, description });

    const call = await contract.hyperionProposal.estimateGas(title, description, msg, "1000000000000000000", {
      gasPrice: 50000000000,
      gasLimit: 5000000
    });
    console.log('call: ', call);
    
    // const tx = await contract.hyperionProposal(title, description, msg, "1000000000000000000", {
    //   gasPrice: 50000000000,
    //   gasLimit: 5000000
    // });
    // console.log('Transaction envoyée, hash :', tx.hash);

    // const receipt = await tx.wait();
    // console.log('Transaction confirmée dans le bloc :', receipt.blockNumber);

    console.log('Proposition soumise avec succès !');
  } catch (error) {
    console.error('Erreur lors de la soumission de la proposition :', error);
  }
}

async function modularProposal({ title, description, msg, proposalType }) {
  const abi = JSON.parse(fs.readFileSync('../helios-chain/precompiles/gov/abi.json').toString()).abi;
  const contract = new ethers.Contract("0x0000000000000000000000000000000000000805", abi, wallet);

  try {
    console.log('Ajout d\'une nouvelle proposition au consensus...');
    console.log('Arguments envoyés au contrat :', { title, description });

    const call = await contract.modularProposal.estimateGas(title, description, msg, "1000000000000000000", proposalType);
    console.log('call: ', call);
    
    const tx = await contract.modularProposal(title, description, msg, "1000000000000000000", proposalType, {
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

async function updateChainSmartContractProposal(chainId, bridgeContractAddress, bridgeContractStartHeight, contractSourceHash) {
  await hyperionProposal({
    title: "clean all batches and txs chain id " + chainId,
    description: "clean all batches and txs chain id " + chainId,
    msg: JSON.stringify({
      "@type": "/helios.hyperion.v1.MsgUpdateChainSmartContract",
      "chain_id": chainId,
      "bridge_contract_address": bridgeContractAddress,
      "bridge_contract_start_height": bridgeContractStartHeight,
      "contract_source_hash": contractSourceHash,
      "first_orchestrator_address": wallet.address,
      "signer": wallet.address
    })
  })
}

async function setPauseProposal(chainId) {
  await hyperionProposal({
    title: `Pause Chain Hyperion ${chainId}`,
    description: `Pause Chain Hyperion ${chainId}`,
    msg: JSON.stringify({
      "@type": "/helios.hyperion.v1.MsgPauseChain",
      "chain_id": chainId,
      "signer": wallet.address
    })
  })
}

async function setUnpauseProposal(chainId) {
  await hyperionProposal({
    title: `Unpause Chain Hyperion ${chainId}`,
    description: `Unpause Chain Hyperion ${chainId}`,
    msg: JSON.stringify({
      "@type": "/helios.hyperion.v1.MsgUnpauseChain",
      "chain_id": chainId,
      "signer": wallet.address
    })
  })
}

async function setWhitelistedAddressesProposal(hyperionId, addresses) {
  await hyperionProposal({
    title: `Set Whitelisted Addresses Hyperion ${hyperionId}`,
    description: `Set Whitelisted Addresses Hyperion ${hyperionId}`,
    msg: JSON.stringify({
      "@type": "/helios.hyperion.v1.MsgSetWhitelistedAddresses",
      "hyperion_id": hyperionId,
      "addresses": addresses,
      "signer": wallet.address
    })
  })
}

async function addOneWhitelistedAddressProposal(hyperionId, address) {
  await hyperionProposal({
    title: `Add One Whitelisted Address Hyperion ${hyperionId}`,
    description: `Add One Whitelisted Address Hyperion ${hyperionId}`,
    msg: JSON.stringify({
      "@type": "/helios.hyperion.v1.MsgAddOneWhitelistedAddress",
      "hyperion_id": hyperionId,
      "address": address,
      "signer": wallet.address
    })
  })
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

async function addCounterpartyChainProposal() {
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
}

async function setTokenToChainProposal(chainId) {
  await hyperionProposal({
    title: `Set Token To Chain Hyperion ${chainId}`,
    description: `Set Token To Chain Hyperion ${chainId}`,
    msg: JSON.stringify({
      "@type": "/helios.hyperion.v1.MsgSetTokenToChain",
      "chain_id": chainId,
      "token_address": "0x47Fa8b2c03bb62aA9e7F1a931536b4629785D123",
      "denom": "ahelios",
      "symbol": "HLS",
      "decimals": 18,
      "is_cosmos_originated": true,
      "is_concensus_token": true,
      "signer": wallet.address
    })
  })
}

async function removeTokenFromChainProposal(chainId) {
  await hyperionProposal({
    title: `Remove Token From Chain Hyperion ${chainId}`,
    description: `Remove Token From Chain Hyperion ${chainId}`,
    msg: JSON.stringify({
      "@type": "/helios.hyperion.v1.MsgRemoveTokenFromChain",
      "chain_id": chainId,
      "denom": "ahelios",
      "signer": wallet.address
    })
  })
}

async function cleanAllSkippedTxsProposal() {
  await hyperionProposal({
    title: `Clean All Skipped Txs`,
    description: `Clean All Skipped Txs`,
    msg: JSON.stringify({
      "@type": "/helios.hyperion.v1.MsgCleanAllSkippedTxs",
      "signer": wallet.address
    })
  })
}

async function main() {

  // await setPauseProposal(97);

  // await setUnpauseProposal(97);
  // await setUnpauseProposal(11155111);


  // await updateChainSmartContractProposal(97, "0xB8ed88AcD8b7ac80d9f546F4D75F33DD19dD5746", 58396704, "0x0000000000000000000000000000000000000000000000000000000000000000");
  // await vote(104316);
  // await vote(104933);

  // await addOneWhitelistedAddressProposal(97, "0x7e62c5e7Eba41fC8c25e605749C476C0236e0604");
  // await addOneWhitelistedAddressProposal(42161, "0x7e62c5e7Eba41fC8c25e605749C476C0236e0604");

  // await setTokenToChainProposal(11155111);

  // await removeTokenFromChainProposal(11155111);

  // await modularProposal({
  //   title: "Update slashing params",
  //   description: "Update slashing params",
  //   msg: JSON.stringify({
  //     "@type": "/cosmos.slashing.v1beta1.MsgUpdateParams",
  //     "params": {
  //       // "@type": "/cosmos.slashing.v1beta1.Params", / not neccessary to define the type of childs protos
  //       "signedBlocksWindow": 1000,
  //       "minSignedPerWindow": "0.2",
  //       "downtimeJailDuration": "600s",
  //       "slashFractionDoubleSign": "0.05",
  //       "slashFractionDowntime": "0.01",
  //     },
  //     "authority": wallet.address
  //   }),
  //   proposalType: "slashing",
  //   initialDepositAmount: "1000000000000000000"
  // });

  cleanAllSkippedTxsProposal();
}

main();