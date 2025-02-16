const ethers = require('ethers');
const WebSocket = require('ws');

const RPC_URL = 'http://localhost:8545';
const COSMOS_RPC_WS = 'ws://localhost:26657/websocket'; // WebSocket Cosmos RPC

const PRIVATE_KEY = '2c37c3d09d7a1c957f01ad200cec69bc287d0a9cc85b4dce694611a4c9c24036';

const PRIVATE_KEY2 = 'e1ab51c450698b0af4722e074e39394bd99822f0b00f1a787a131b48c14d4483'

const PRECOMPILE_CONTRACT_ADDRESS = '0x0000000000000000000000000000000000000806';

const provider = new ethers.JsonRpcProvider(RPC_URL);

const wallet = new ethers.Wallet(PRIVATE_KEY, provider);
const wallet2 = new ethers.Wallet(PRIVATE_KEY2, provider);


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

const addNewProposal = [
  {
    "inputs": [
      { "internalType": "address", "name": "delegatorAddress", "type": "address" },
      { "internalType": "string", "name": "validatorAddress", "type": "string" },
      { "internalType": "uint256", "name": "amount", "type": "uint256" },
      { "internalType": "string", "name": "denom", "type": "string" }
    ],
    "name": "delegate",
    "outputs": [{ "internalType": "bool", "name": "success", "type": "bool" }],
    "stateMutability": "nonpayable",
    "type": "function"
  }
];

const unstakingAbi = [
  {
    "inputs": [
      { "internalType": "address", "name": "delegatorAddress", "type": "address" },
      { "internalType": "string", "name": "validatorAddress", "type": "string" },
      { "internalType": "uint256", "name": "amount", "type": "uint256" },
      { "internalType": "string", "name": "denom", "type": "string" }
    ],
    "name": "undelegate",
    "outputs": [{ "internalType": "bool", "name": "success", "type": "bool" }],
    "stateMutability": "nonpayable",
    "type": "function"
  }
];

const proposalAbi = [
  {
    "inputs": [
      { "internalType": "string", "name": "title", "type": "string" },
      { "internalType": "string", "name": "description", "type": "string" },
      {
        "components": [
          { "internalType": "string", "name": "denom", "type": "string" },
          { "internalType": "string", "name": "contractAddress", "type": "string" },
          { "internalType": "string", "name": "chainId", "type": "string" },
          { "internalType": "uint32", "name": "decimals", "type": "uint32" },
          { "internalType": "uint64", "name": "baseWeight", "type": "uint64" },
          { "internalType": "string", "name": "metadata", "type": "string" }
        ],
        "internalType": "struct Asset[]",
        "name": "assets",
        "type": "tuple[]"
      },
      {
        "internalType": "uint256",
        "name": "initialDepositAmount",
        "type": "uint256"
      }
    ],
    "name": "addNewAssetProposal",
    "outputs": [
      { "internalType": "uint64", "name": "proposalId", "type": "uint64" }
    ],
    "stateMutability": "nonpayable",
    "type": "function"
  }
];

const updateProposalAbi = [
  {
    "inputs": [
      {
        "internalType": "string",
        "name": "title",
        "type": "string"
      },
      {
        "internalType": "string",
        "name": "description",
        "type": "string"
      },
      {
        "components": [
          { "internalType": "string", "name": "denom", "type": "string" },
          { "internalType": "string", "name": "magnitude", "type": "string" },
          { "internalType": "string", "name": "direction", "type": "string" }
        ],
        "internalType": "struct WeightUpdateData[]",
        "name": "updates",
        "type": "tuple[]"
      },
      {
        "internalType": "uint256",
        "name": "initialDepositAmount",
        "type": "uint256"
      }
    ],
    "name": "updateAssetProposal",
    "outputs": [
      {
        "internalType": "uint64",
        "name": "proposalId",
        "type": "uint64"
      }
    ],
    "stateMutability": "nonpayable",
    "type": "function"
  }
]


const contract = new ethers.Contract(PRECOMPILE_CONTRACT_ADDRESS, abi, wallet);

const tokenName = 'USDTeeeee';
const tokenSymbol = 'USDTeeeee';
const tokenTotalSupply = ethers.parseUnits('5000000', 18);
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

async function fetch(){
const contractAddress = '0xD4949664cD82660AaE99bEdc034a0deA8A0bd517';

const abi = [
  'function name() view returns (string)',
  'function symbol() view returns (string)',
  'function decimals() view returns (uint8)',
  'function totalSupply() view returns (uint256)',
  'function balanceOf(address account) view returns (uint256)',
];

// Créer une instance du contrat
const contract = new ethers.Contract(contractAddress, abi, provider);
const name = await contract.name();

}

async function delegate() {

  const validatorAddress = 'heliosvaloper1zun8av07cvqcfr2t29qwmh8ufz29gfat770rla'; // Adresse Cosmos du validateur
  const delegateAmount = ethers.parseUnits('10', 18); // Montant à déléguer (ex: 10 tokens)
    // Extraire et afficher la clé publique
    console.log("wallet : ", wallet.address)
  try {
    console.log('Délégation en cours...');

    const contract = new ethers.Contract('0x0000000000000000000000000000000000000800', addNewProposal, wallet);
    const tx = await contract.delegate(wallet.address, validatorAddress, delegateAmount, "WETH");
    console.log('Transaction envoyée, hash :', tx.hash);

    const receipt = await tx.wait();
    console.log('Transaction confirmée dans le bloc :', receipt.blockNumber);

    console.log('Délégation réussie !');
  } catch (error) {
    console.error('Erreur lors de la délégation :', error);
  }
}

async function addNewConsensusProposal() {
  const contract = new ethers.Contract("0x0000000000000000000000000000000000000805", proposalAbi, wallet2);

  const title = 'Whitelist WETH into the consensus with a base stake of power 100';
  const description = 'Explaining why WETH would be a good potential for Helios consensus and why it would secure the market';
  const assets = [
    {
      denom: 'WETH',
      contractAddress: '0x80b5a32E4F032B2a058b4F29EC95EEfEEB87aDcd', // Exact match to ABI
      chainId: 'ethereum',                                          // Exact match to ABI
      decimals: 6,
      baseWeight: 100,
      metadata: 'WETH stablecoin'
    }
  ];

  try {
    console.log('Ajout d\'une nouvelle proposition au consensus...');
    console.log('Arguments envoyés au contrat :', { title, description, assets });
    
    const tx = await contract.addNewAssetProposal(title, description, assets, "1000000000000000000");
    console.log('Transaction envoyée, hash :', tx.hash);

    const receipt = await tx.wait();
    console.log('Transaction confirmée dans le bloc :', receipt.blockNumber);

    console.log('Proposition soumise avec succès !');
  } catch (error) {
    console.error('Erreur lors de la soumission de la proposition :', error);
  }
}

async function updateConsensusProposal() {
  const contract = new ethers.Contract("0x0000000000000000000000000000000000000805", updateProposalAbi, wallet2);

  const title = 'Update WETH to higt';
  const description = 'update WETH';
  const updates = [
    {
      denom: 'WETH',
      magnitude: 'high',
      direction: 'down'
    }
  ];

  try {
    console.log('update proposition au consensus...');
    console.log('Arguments envoyés au contrat :', { title, description, updates });
    
    const tx = await contract.updateAssetProposal(title, description, updates, "1000000000000000000");
    console.log('Transaction envoyée, hash :', tx.hash);

    const receipt = await tx.wait();
    console.log('Transaction confirmée dans le bloc :', receipt.blockNumber);

    console.log('Proposition soumise avec succès !');
  } catch (error) {
    console.error('Erreur lors de la soumission de la proposition :', error);
  }
}

const voteAbi = [
  {
    "inputs": [
      {
        "internalType": "address",
        "name": "voter",
        "type": "address"
      },
      {
        "internalType": "uint64",
        "name": "proposalId",
        "type": "uint64"
      },
      {
        "internalType": "enum VoteOption",
        "name": "option",
        "type": "uint8"
      },
      {
        "internalType": "string",
        "name": "metadata",
        "type": "string"
      }
    ],
    "name": "vote",
    "outputs": [
      {
        "internalType": "bool",
        "name": "success",
        "type": "bool"
      }
    ],
    "stateMutability": "nonpayable",
    "type": "function"
  },
];

async function vote(){
  const contract = new ethers.Contract("0x0000000000000000000000000000000000000805", voteAbi, wallet);

  try {
    console.log('Ajout d\'une nouvelle proposition au consensus...');
    
    const tx = await contract.vote(wallet.address, 4, 1, "voting testtest");
    console.log('Transaction envoyée, hash :', tx.hash);

    const receipt = await tx.wait();
    console.log('Transaction confirmée dans le bloc :', receipt.blockNumber);

    console.log('vote fait a success soumise avec succès !');
  } catch (error) {
    console.error('Erreur lors de la soumission de la proposition :', error);
  }
}

async function undelegate() {

  const validatorAddress = 'heliosvaloper1zun8av07cvqcfr2t29qwmh8ufz29gfat770rla'; // Adresse Cosmos du validateur
  const delegateAmount = ethers.parseUnits('10', 18); // Montant à déléguer (ex: 10 tokens)
    // Extraire et afficher la clé publique
    console.log("wallet : ", wallet.address)
  try {
    console.log('UnDélégation en cours...');

    const contract = new ethers.Contract('0x0000000000000000000000000000000000000800', unstakingAbi, wallet);
    const tx = await contract.undelegate(wallet.address, validatorAddress, delegateAmount, "WETH");
    console.log('Transaction envoyée, hash :', tx.hash);

    const receipt = await tx.wait();
    console.log('Transaction confirmée dans le bloc :', receipt.blockNumber);

    console.log('UnDélégation réussie !');
  } catch (error) {
    console.error('Erreur lors de la délégation :', error);
  }
}

async function main() {
  await create();
  //await fetch();
  //await delegate();
  //await addNewConsensusProposal();
  //await updateConsensusProposal();
  //await vote();
  //await undelegate();
  
}

main();