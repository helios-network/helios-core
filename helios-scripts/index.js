const ethers = require('ethers');
const WebSocket = require('ws');

const RPC_URL = 'http://localhost:8545';
const COSMOS_RPC_WS = 'ws://localhost:26657/websocket'; // WebSocket Cosmos RPC

const PRIVATE_KEY = '2c37c3d09d7a1c957f01ad200cec69bc287d0a9cc85b4dce694611a4c9c24036';

const PRIVATE_KEY2 = '262d4b734eae5e891f226b43bd893a97449b9f4df793ac505001c03575100700'

const PRIVATE_KEYS3 = '9eaba88611d3b24507fd077e609d18bed9b0e912f46b9f51f389905aad428875'
const PUBLIC_KEY = '0x17267eB1FEC301848d4B5140eDDCFC48945427Ab'
const PUBLIC_KEY_2 = '0xc9728bFb36F8D2f9d39a5e7ce19AA11aF27dB440'


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

const tokenName = 'WETH';
const tokenSymbol = 'WETH';
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
const name = await contract.balanceOf('0x17267eB1FEC301848d4B5140eDDCFC48945427Ab')
console.log(name)

}

async function delegate() {

  const validatorAddress = wallet.address; // Adresse du validateur
  const delegateAmount = ethers.parseUnits('0.5', 18); // Montant à déléguer (ex: 10 tokens)
    // Extraire et afficher la clé publique
    console.log("wallet : ", wallet.address)
  try {
    console.log('Délégation en cours...');

    const contract = new ethers.Contract('0x0000000000000000000000000000000000000800', delegateAbi, wallet);
    const tx = await contract.delegate(wallet.address, validatorAddress, delegateAmount, "ahelios");
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
    
    const tx = await contract.vote(wallet.address, 5, 1, "voting testtest");
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
  const delegateAmount = ethers.parseUnits('100', 18); // Montant à déléguer (ex: 10 tokens)
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

const createValidatorAbi = [
  {
    "inputs": [
      {
        "components": [
          {
            "internalType": "string",
            "name": "moniker",
            "type": "string"
          },
          {
            "internalType": "string",
            "name": "identity",
            "type": "string"
          },
          {
            "internalType": "string",
            "name": "website",
            "type": "string"
          },
          {
            "internalType": "string",
            "name": "securityContact",
            "type": "string"
          },
          {
            "internalType": "string",
            "name": "details",
            "type": "string"
          }
        ],
        "internalType": "struct Description",
        "name": "description",
        "type": "tuple"
      },
      {
        "components": [
          {
            "internalType": "uint256",
            "name": "rate",
            "type": "uint256"
          },
          {
            "internalType": "uint256",
            "name": "maxRate",
            "type": "uint256"
          },
          {
            "internalType": "uint256",
            "name": "maxChangeRate",
            "type": "uint256"
          }
        ],
        "internalType": "struct CommissionRates",
        "name": "commissionRates",
        "type": "tuple"
      },
      {
        "internalType": "uint256",
        "name": "minSelfDelegation",
        "type": "uint256"
      },
      {
        "internalType": "address",
        "name": "validatorAddress",
        "type": "address"
      },
      {
        "internalType": "string",
        "name": "pubkey",
        "type": "string"
      },
      {
        "internalType": "uint256",
        "name": "value",
        "type": "uint256"
      },
      {
        "internalType": "uint256",
        "name": "minDelegation",
        "type": "uint256"
      }
    ],
    "name": "createValidator",
    "outputs": [
      {
        "internalType": "bool",
        "name": "success",
        "type": "bool"
      }
    ],
    "stateMutability": "nonpayable",
    "type": "function"
  }
];


async function createValidator() {

  const createValidatorContract = new ethers.Contract(
    '0x0000000000000000000000000000000000000800',
    createValidatorAbi,
    wallet2
  );


  const description = {
    moniker:          "MyNode",
    identity:         "",
    website:          "https://mynode.example",
    securityContact:  "mynode@example.com",
    details:          "This is my great node"
  };

  const commissionRates = {
    rate:          ethers.parseUnits("0.05", 18),  // 5%
    maxRate:       ethers.parseUnits("0.20", 18),  // 20%
    maxChangeRate: ethers.parseUnits("0.01", 18),  // 1%
  };

  const minSelfDelegation = "1";
  const validatorAddress  = wallet2.address; 
  // heliades tendermint show-validator
  const pubkey            = "D5zf7fEgixSNzi9wdaK009cyMU6cn7mcmA8IpqtVSSc=";
  const value             = ethers.parseUnits("10", 18);

  try {
    // 4) Envoyer la transaction
    console.log("Envoi de createValidator...");
    const tx = await createValidatorContract.createValidator(
      description,
      commissionRates,
      minSelfDelegation,
      wallet2.address,
      pubkey,
      value,
      0
    );
    console.log("Tx envoyée. hash =", tx.hash);

    const receipt = await tx.wait();
    console.log("Transaction confirmée dans le bloc :", receipt.blockNumber);

    console.log("Validateur créé avec succès !");
  } catch (err) {
    console.error("Erreur lors de la création du validateur :", err);
  }
}


const editValidatorAbi = [
  {
    "inputs": [
      {
        "components": [
          { "internalType": "string", "name": "moniker", "type": "string" },
          { "internalType": "string", "name": "identity", "type": "string" },
          { "internalType": "string", "name": "website", "type": "string" },
          { "internalType": "string", "name": "securityContact", "type": "string" },
          { "internalType": "string", "name": "details", "type": "string" }
        ],
        "internalType": "struct Description",
        "name": "description",
        "type": "tuple"
      },
      { "internalType": "address", "name": "validatorAddress", "type": "address" },
      { "internalType": "int256", "name": "commissionRate", "type": "int256" },
      { "internalType": "int256", "name": "minSelfDelegation", "type": "int256" }
    ],
    "name": "editValidator",
    "outputs": [{ "internalType": "bool", "name": "success", "type": "bool" }],
    "stateMutability": "nonpayable",
    "type": "function"
  }
];


async function editValidator() {
  const editValidatorContract = new ethers.Contract(
    '0x0000000000000000000000000000000000000800', // Adresse du precompile staking
    editValidatorAbi,
    wallet
  );

  const description = {
    moniker: "UpdatedNode",
    identity: "",
    website: "https://updatednode.example",
    securityContact: "newcontact@example.com",
    details: "Updated validator details"
  };

  const validatorAddress = wallet.address;

  // Si tu ne veux PAS modifier commissionRate ou minSelfDelegation, utilise -1.
  const commissionRate = ethers.parseUnits("0.08", 18);  // Modifier à 8%, sinon -1 pour ne pas changer
  const minSelfDelegation = -1;  // Mettre -1 pour ne pas modifier cette valeur

  try {
    console.log("Envoi de editValidator...");
    const tx = await editValidatorContract.editValidator(
      description,
      validatorAddress,
      commissionRate,
      minSelfDelegation
    );

    console.log("Transaction envoyée, hash:", tx.hash);

    const receipt = await tx.wait();
    console.log("Transaction confirmée dans le bloc:", receipt.blockNumber);

    console.log("Modification du validateur réussie !");
  } catch (err) {
    console.error("Erreur lors de la modification du validateur:", err);
  }
}

const redelegateAbi = [
  {
    "inputs": [
      {
        "internalType": "address",
        "name": "delegatorAddress",
        "type": "address"
      },
      {
        "internalType": "address",
        "name": "validatorSrcAddress",
        "type": "address"
      },
      {
        "internalType": "address",
        "name": "validatorDstAddress",
        "type": "address"
      },
      {
        "internalType": "uint256",
        "name": "amount",
        "type": "uint256"
      },
      {
        "internalType": "string",
        "name": "denom",
        "type": "string"
      }
    ],
    "name": "redelegate",
    "outputs": [
      {
        "internalType": "int64",
        "name": "completionTime",
        "type": "int64"
      }
    ],
    "stateMutability": "nonpayable",
    "type": "function"
  },
]

async function redelegate() {
  // On réutilise l'adresse de ton precompile Staking
  const contractAddress = '0x0000000000000000000000000000000000000800';

  // Charger l'ABI juste créée
  const contract = new ethers.Contract(contractAddress, redelegateAbi, wallet);

  // Paramètres
  // Remplace par tes vraies adresses et valeurs
  const delegatorAddress    = wallet.address;
  const validatorSrcAddress =  wallet2.address;
  const validatorDstAddress =  wallet.address;
  const amount              = ethers.parseUnits('100', 18);
  const denom               = "WETH"; //

  try {
    console.log("Redelegation en cours...");

    const tx = await contract.redelegate(
      delegatorAddress,
      validatorSrcAddress,
      validatorDstAddress,
      amount,
      denom
    );

    console.log("Transaction envoyée, hash:", tx.hash);

    // Attendre la confirmation
    const receipt = await tx.wait();
    console.log("Transaction confirmée dans le bloc:", receipt.blockNumber);

    // Selon l’implémentation du contrat, on peut récupérer `completionTime`
    // via `receipt.events`, si l’event l’envoie. Sinon, la fonction renvoie
    // la valeur directe.
    console.log("Rédelegation réussie !");
  } catch (error) {
    console.error("Erreur lors de la rédelegation:", error);
  }
}



async function main() {
  // await create();
  // await fetch();
  await delegate();
  //await addNewConsensusProposal();
  //await updateConsensusProposal();
  //await vote();
  //await undelegate();
  //await getRewards();
  
  // await createValidator();
  //await editValidator();
  //await redelegate();
 
}

main();