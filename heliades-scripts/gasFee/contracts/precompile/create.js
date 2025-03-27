
const ethers = require('ethers');
const WebSocket = require('ws');
const fs = require('fs');

const RPC_URL = 'http://localhost:8545';
const COSMOS_RPC_WS = 'ws://localhost:26657/websocket'; // WebSocket Cosmos RPC

const PRECOMPILE_CONTRACT_ADDRESS = '0x0000000000000000000000000000000000000806';
const PRIVATE_KEY = '2c37c3d09d7a1c957f01ad200cec69bc287d0a9cc85b4dce694611a4c9c24036';

const provider = new ethers.JsonRpcProvider(RPC_URL);

const wallet = new ethers.Wallet(PRIVATE_KEY, provider);

const abi = [
	{
	  "inputs": [
		{ "internalType": "string", "name": "name", "type": "string" },
		{ "internalType": "string", "name": "symbol", "type": "string" },
		{ "internalType": "uint256", "name": "totalSupply", "type": "uint256" },
		{ "internalType": "uint8", "name": "decimals", "type": "uint8" }
	  ],
	  "name": "createErc20",
	  "outputs": [
		{ "internalType": "address", "name": "tokenAddress", "type": "address" }
	  ],
	  "stateMutability": "nonpayable",
	  "type": "function"
	}
  ];

const contract = new ethers.Contract(PRECOMPILE_CONTRACT_ADDRESS, abi, wallet);

const tokenName = 'BNBFDP1';
const tokenSymbol = 'BNBFDP1';
const tokenTotalSupply = ethers.parseUnits('100', 18);
const tokenDecimals = 18;

async function create(){
  try {
    console.log('Création du token ERC20...');
    
    const tx = await contract.createErc20(tokenName, tokenSymbol, tokenTotalSupply, tokenDecimals);
    console.log('Transaction envoyée, hash :', tx.hash);

    const receipt = await tx.wait();
    console.log('Transaction confirmée dans le bloc :', receipt.blockNumber);

    receipt.logs.forEach(log => {
      try {
        const parsedLog = contract.interface.parseLog(log);
        if (parsedLog.name === 'ERC20Created') {
          console.log('Token créé avec succès :');
          console.log('Créateur :', parsedLog.args.creator);
          console.log('Adresse du token :', parsedLog.args.tokenAddress);
          console.log('Nom du token :', parsedLog.args.name);
          console.log('Symbole du token :', parsedLog.args.symbol);
        }
      } catch (err) {
      }
    });
  } catch (error) {
    console.error('Une erreur est survenue :', error);
  }
}

create();