const ethers = require('ethers');
const WebSocket = require('ws');
const fs = require('fs');

const RPC_HELIOS_URL = 'http://localhost:8545';
const PRIVATE_KEY = '';
const provider = new ethers.JsonRpcProvider(RPC_HELIOS_URL);
const walletConnectedToHelios = new ethers.Wallet(PRIVATE_KEY, provider);

async function modularProposal({ title, description, msg, proposalType }) {
  const abi = JSON.parse(fs.readFileSync('../helios-chain/precompiles/gov/abi.json').toString()).abi;
  const contract = new ethers.Contract("0x0000000000000000000000000000000000000805", abi, walletConnectedToHelios);

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

async function vote(proposalId){
  const abi = JSON.parse(fs.readFileSync('../helios-chain/precompiles/gov/abi.json').toString()).abi;
  const contract = new ethers.Contract("0x0000000000000000000000000000000000000805", abi, walletConnectedToHelios);

  try {
    console.log('Ajout d\'une nouvelle proposition au consensus...');
    
    const tx = await contract.vote(walletConnectedToHelios.address, proposalId, 1, "voting testtest");
    console.log('Transaction envoyée, hash :', tx.hash);

    const receipt = await tx.wait();
    console.log('Transaction confirmée dans le bloc :', receipt.blockNumber);

    console.log('vote fait a success soumise avec succès !');
  } catch (error) {
    console.error('Erreur lors de la soumission de la proposition :', error);
  }
}

async function main() {
  await modularProposal({
    title: "Update slashing params",
    description: "Update slashing params",
    msg: JSON.stringify({
      "@type": "/cosmos.upgrade.v1beta1.MsgSoftwareUpgrade",
      "plan": {
          "name": "v0.0.272",
          "height": 35,
          "info": JSON.stringify({
            "version": "v0.0.272",
            "size": 200698690,
            "hash": "f52c9a2ee2f5d118d1b2663f635dc7935c65d47fc00cd0c6338845870669491e"
          })
      },
      "authority": walletConnectedToHelios.address,
    }),
    proposalType: "upgrade",
    initialDepositAmount: "1000000000000000000"
  });
  await vote(1);
}

main();