const ethers = require('ethers');
const WebSocket = require('ws');
const fs = require('fs');

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

const delegateAbi = [
  {
    "inputs": [
      { "internalType": "address", "name": "delegatorAddress", "type": "address" },
      { "internalType": "address", "name": "validatorAddress", "type": "address" },
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
      { "internalType": "address", "name": "validatorAddress", "type": "address" },
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
];

const withdrawDelegatorRewardsAbi = [
  {
    "inputs": [
      {
        "internalType": "address",
        "name": "delegatorAddress",
        "type": "address"
      },
      {
        "internalType": "address",
        "name": "validatorAddress",
        "type": "address"
      }
    ],
    "name": "withdrawDelegatorRewards",
    "outputs": [
      {
        "components": [
          {
            "internalType": "string",
            "name": "denom",
            "type": "string"
          },
          {
            "internalType": "uint256",
            "name": "amount",
            "type": "uint256"
          }
        ],
        "internalType": "struct Coin[]",
        "name": "amount",
        "type": "tuple[]"
      }
    ],
    "stateMutability": "nonpayable",
    "type": "function"
  }
]


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

async function fetch(){
  const contractAddress = '0x80b5a32E4F032B2a058b4F29EC95EEfEEB87aDcd';

  const abi = [
    'function name() view returns (string)',
    'function symbol() view returns (string)',
    'function decimals() view returns (uint8)',
    'function totalSupply() view returns (uint256)',
    'function balanceOf(address account) view returns (uint256)',
  ];

  // Créer une instance du contrat
  const contract = new ethers.Contract(contractAddress, abi, provider);
  
  // Récupérer les informations du token
  const name = await contract.name();
  const symbol = await contract.symbol();
  const decimals = await contract.decimals();
  const totalSupply = await contract.totalSupply();
  const balance = await contract.balanceOf(wallet.address); // Assuming you want the balance of the wallet

  // Afficher les informations
  console.log('Token Name:', name);
  console.log('Token Symbol:', symbol);
  console.log('Token Decimals:', decimals);
  console.log('Total Supply:', totalSupply.toString());
  console.log('Balance of Wallet:', balance.toString());
}

async function delegate() {

  const validatorAddress = '0x17267eb1fec301848d4b5140eddcfc48945427ab'; // Adresse du validateur
  const delegateAmount = ethers.parseUnits('1.5', 18); // Montant à déléguer (ex: 10 tokens)
    // Extraire et afficher la clé publique
    console.log("wallet : ", wallet.address)
  try {
    console.log('Délégation en cours...');

    const contract = new ethers.Contract('0x0000000000000000000000000000000000000800', delegateAbi, wallet);
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
    
    const tx = await contract.addNewAssetProposal(title, description, assets, "1000000000000000000", {
      gasPrice: 50000000000,
      gasLimit: 500000
    });
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
      denom: 'BNB',
      magnitude: 'high',
      direction: 'up'
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
    
    const tx = await contract.vote(wallet.address, 2, 1, "voting testtest");
    console.log('Transaction envoyée, hash :', tx.hash);

    const receipt = await tx.wait();
    console.log('Transaction confirmée dans le bloc :', receipt.blockNumber);

    console.log('vote fait a success soumise avec succès !');
  } catch (error) {
    console.error('Erreur lors de la soumission de la proposition :', error);
  }
}

async function undelegate() {

  const validatorAddress = '0x17267eb1fec301848d4b5140eddcfc48945427ab'; // Adresse du validateur
  const delegateAmount = ethers.parseUnits('0.1', 18); // Montant à déléguer (ex: 10 tokens)
    // Extraire et afficher la clé publique
    console.log("wallet : ", wallet.address)
  try {
    console.log('UnDélégation en cours...');

    const contract = new ethers.Contract('0x0000000000000000000000000000000000000800', unstakingAbi, wallet);
    const tx = await contract.undelegate(wallet.address, validatorAddress, delegateAmount, "BNB");
    console.log('Transaction envoyée, hash :', tx.hash);

    const receipt = await tx.wait();
    console.log('Transaction confirmée dans le bloc :', receipt.blockNumber);

    console.log('UnDélégation réussie !');
  } catch (error) {
    console.error('Erreur lors de la délégation :', error);
  }
}

async function getRewards() {

  const validatorAddress = '0x17267eb1fec301848d4b5140eddcfc48945427ab'; // Adresse Cosmos du validateur
  const delegateAmount = ethers.parseUnits('10', 18); // Montant à déléguer (ex: 10 tokens)
    // Extraire et afficher la clé publique
    console.log("wallet : ", wallet.address)
  try {
    console.log('WithdrawRewards en cours...');

    const contract = new ethers.Contract('0x0000000000000000000000000000000000000801', withdrawDelegatorRewardsAbi, wallet);
    const tx = await contract.withdrawDelegatorRewards.staticCall(wallet.address, validatorAddress, {
      gasLimit: 500000,
      gasPrice: ethers.parseUnits('20', "gwei")
    });
    console.log(tx);
    // console.log('Transaction envoyée, hash :', tx.hash);

    // const receipt = await tx.wait();
    // console.log('Transaction confirmée dans le bloc :', receipt.blockNumber);

    // console.log('WithdrawRewards réussie !');
  } catch (error) {
    console.error('Erreur lors de la WithdrawRewards :', error);
  }
}

async function getRewards() {

  const validatorAddress = '0x17267eb1fec301848d4b5140eddcfc48945427ab'; // Adresse Cosmos du validateur
  const delegateAmount = ethers.parseUnits('10', 18); // Montant à déléguer (ex: 10 tokens)
    // Extraire et afficher la clé publique
    console.log("wallet : ", wallet.address)
  try {
    console.log('WithdrawRewards en cours...');

    const contract = new ethers.Contract('0x0000000000000000000000000000000000000801', withdrawDelegatorRewardsAbi, wallet);
    const tx = await contract.withdrawDelegatorRewards(wallet.address, validatorAddress);
    console.log('Transaction envoyée, hash :', tx.hash);

    const receipt = await tx.wait();
    console.log('Transaction confirmée dans le bloc :', receipt.blockNumber);

    console.log('WithdrawRewards réussie !');
  } catch (error) {
    console.error('Erreur lors de la WithdrawRewards :', error);
  }
}

async function addCounterpartyChainParams() {
  console.log("wallet : ", wallet.address)
  try {
    console.log('addCounterpartyChainParams en cours...');

    const hyperionAbi = JSON.parse(fs.readFileSync('../helios-chain/precompiles/hyperion/abi.json').toString()).abi;
    const contract = new ethers.Contract('0x0000000000000000000000000000000000000900', hyperionAbi, wallet);
    const tx = await contract.addCounterpartyChainParams(
      21, // New HyperionId
      "", // bridge contract hash
      "0x17AAd46782B95fbDE4ff1051397F24e554f6A487", // bridge contract address
      80002, // chainId
      19437427 // start height
    );
    console.log('Transaction envoyée, hash :', tx.hash);

    const receipt = await tx.wait();
    console.log('Transaction confirmée dans le bloc :', receipt.blockNumber);

    console.log(receipt);
  } catch (error) {
    console.error('Erreur lors de la addCounterpartyChainParams :', error);
  }
}

async function setOrchestratorAddresses() {
  console.log("wallet : ", wallet.address)
  try {
    console.log('setOrchestratorAddresses en cours...');

    const chronosAbi = JSON.parse(fs.readFileSync('../helios-chain/precompiles/hyperion/abi.json').toString()).abi;
    const contract = new ethers.Contract('0x0000000000000000000000000000000000000900', chronosAbi, wallet);
    const tx = await contract.setOrchestratorAddresses( // I'm validator and
      "0x17267eB1FEC301848d4B5140eDDCFC48945427Ab", // address of my hyperion Validator
      21 // HyperionId
    );
    console.log('Transaction envoyée, hash :', tx.hash);

    const receipt = await tx.wait();
    console.log('Transaction confirmée dans le bloc :', receipt.blockNumber);

    console.log(receipt);
  } catch (error) {
    console.error('Erreur lors de la setOrchestratorAddresses :', error);
  }
}

async function sendToChain() {
  console.log("wallet : ", wallet.address)
  try {
    console.log('sendToChain en cours...');

    const hyperionAbi = JSON.parse(fs.readFileSync('../helios-chain/precompiles/hyperion/abi.json').toString()).abi;
    const contract = new ethers.Contract('0x0000000000000000000000000000000000000900', hyperionAbi, wallet);
    const tx = await contract.sendToChain( // I'm validator and
      80002, // chainId
      "0x17267eB1FEC301848d4B5140eDDCFC48945427Ab", // receiver
      "0x80b5a32e4f032b2a058b4f29ec95eefeeb87adcd", // token address (ex: USDT)
      ethers.parseEther("33"), // amount to transfer (ex: 10 USDT)
      ethers.parseEther("1"), // fee you want to pay (ex: 1 USDT)
    );
    console.log('Transaction envoyée, hash :', tx.hash);

    const receipt = await tx.wait();
    console.log('Transaction confirmée dans le bloc :', receipt.blockNumber);

    console.log(receipt);
  } catch (error) {
    console.error('Erreur lors de la sendToChain :', error);
  }
}

async function createCron() {
  console.log("wallet : ", wallet.address)
  try {
    console.log('createCron en cours...');

    // let x = await wallet.sendTransaction({
    //   value: ethers.parseEther("500"),
    //   to: wallet2.address
    // });
    // await x.wait();

    const chronosAbi = JSON.parse(fs.readFileSync('../helios-chain/precompiles/chronos/abi.json').toString()).abi;
    const contract = new ethers.Contract('0x0000000000000000000000000000000000000830', chronosAbi, wallet);
    const tx = await contract.createCron(
      "0xa80799d619abafB179e843025571d7913bC00ce8",
      `[ { "inputs": [], "name": "increment", "outputs": [], "stateMutability": "nonpayable", "type": "function" } ]`,
      "increment", // methodName
      [], // params
      1, // frequency
      0, // expirationBlock
      400000, // gasLimit
      ethers.parseUnits("2", "gwei"), // maxGasPrice
      ethers.parseEther("1")
    );
    console.log('Transaction envoyée, hash :', tx.hash);

    const receipt = await tx.wait();
    console.log('Transaction confirmée dans le bloc :', receipt.blockNumber);

    console.log(receipt);
  } catch (error) {
    console.error('Erreur lors de la createCron :', error);
  }
}

async function cancelCron() {
  console.log("wallet : ", wallet.address)
  try {
    console.log('createCron en cours...');

    const chronosAbi = JSON.parse(fs.readFileSync('../helios-chain/precompiles/chronos/abi.json').toString()).abi;
    const contract = new ethers.Contract('0x0000000000000000000000000000000000000830', chronosAbi, wallet);
    const tx = await contract.cancelCron(
      1
    );
    console.log('Transaction envoyée, hash :', tx.hash);

    const receipt = await tx.wait();
    console.log('Transaction confirmée dans le bloc :', receipt.blockNumber);

    console.log(receipt);
  } catch (error) {
    console.error('Erreur lors de la cancelCron :', error);
  }
}

async function getEventsCronCreated() {
  const chronosAbi = JSON.parse(fs.readFileSync('../helios-chain/precompiles/chronos/abi.json').toString()).abi;
  const wsProvider = new ethers.WebSocketProvider('ws://testnet1.helioschainlabs.org:8546');
  const contract = new ethers.Contract('0x0000000000000000000000000000000000000830', chronosAbi, wsProvider);

  contract.on('CronCreated', (from, to, cronId, event) => {
    console.log('New event received!');
    console.log('even:', event);
    console.log('cronId:', cronId.toString());
  });
}

async function getEventsCronCancelled() {
  const chronosAbi = JSON.parse(fs.readFileSync('../helios-chain/precompiles/chronos/abi.json').toString()).abi;
  const wsProvider = new ethers.WebSocketProvider('ws://localhost:8546');
  const contract = new ethers.Contract('0x0000000000000000000000000000000000000830', chronosAbi, wsProvider);

  contract.on('CronCancelled', (from, to, cronId, event) => {
    console.log('New event received!');
    console.log('even:', event);
    console.log('cronId:', cronId.toString());
  });
}

async function getEvents() {
  const abiContract = [
    {
      "anonymous": false,
      "inputs": [
        {
          "indexed": false,
          "internalType": "uint256",
          "name": "newCount",
          "type": "uint256"
        }
      ],
      "name": "CountIncremented",
      "type": "event"
    }
  ];

  const wsProvider = new ethers.WebSocketProvider('ws://localhost:8546');
  const contract = new ethers.Contract('0xEE40f268487f9c2D664Aa66Cf5fD1B01d8b9fC3F', abiContract, wsProvider);
  // const contract = new ethers.Contract('0x8cbF1A9167F66B9B3310Aab56E4fEFc17514d23A', abiContract, wallet);
  // test events
  // Obtenir le bloc actuel
  // const currentBlock = await provider.getBlockNumber();
  // Chercher sur les 1000 derniers blocs par exemple
  // const fromBlock = Math.max(0, currentBlock - 1000);
  
  // const filter = contract.filters.CountIncremented();
  // Spécifier la plage de blocs
  // const events = await contract.queryFilter(filter, fromBlock, currentBlock);

  // const eventSignature = "CountIncremented(uint256)";
  // const eventHash = ethers.id(eventSignature);
  // console.log("Event signature:", eventSignature);
  // console.log("Event hash (keccak256):", eventHash);
  // console.log("Recherche d'événements du bloc", fromBlock, "au bloc", currentBlock);

  // for (const event of events) {
  //   console.log('Event:', event);
  // }

  contract.on('CountIncremented', (newCount, event) => {
    console.log('New event received!');
    console.log('even:', event);
    console.log('New count:', newCount.toString());
  });
}

async function createCronCallBackData() {
  console.log("wallet : ", wallet.address)
  try {
    console.log('createCronCallBackData en cours...');

    // let x = await wallet.sendTransaction({
    //   value: ethers.parseEther("500"),
    //   to: wallet2.address
    // });
    // await x.wait();

    const chronosAbi = JSON.parse(fs.readFileSync('../helios-chain/precompiles/chronos/abi.json').toString()).abi;
    const contract = new ethers.Contract('0x0000000000000000000000000000000000000830', chronosAbi, wallet);
    const tx = await contract.createCallbackConditionedCron(
      "0x6E85bfd36946631d93e0D4a8d6eef2d88e4F4803",
      "callBack", // methodName
      0, // expirationBlock
      400000, // gasLimit
      ethers.parseUnits("2", "gwei"), // maxGasPrice
      ethers.parseEther("1")
    );
    console.log('Transaction envoyée, hash :', tx.hash);

    const receipt = await tx.wait();
    console.log('Transaction confirmée dans le bloc :', receipt.blockNumber);

    console.log(receipt);
  } catch (error) {
    console.error('Erreur lors de la createCron :', error);
  }
}

async function main() {
  // await createCronCallBackData();
  // await createCron();
  // await getEvents();
  // await getEventsCronCancelled();
  // await cancelCron();
  // await getEventsEVMCallScheduled();
  // await create();
  //await fetch();
  //await delegate();
  //await addNewConsensusProposal();
  //await updateConsensusProposal();
  //await vote();
  // await undelegate();

  // await getEventsCronCreated();

  // await getRewards();

  await sendToChain();
  // await setOrchestratorAddresses();
  // await addCounterpartyChainParams();
  
}

main();