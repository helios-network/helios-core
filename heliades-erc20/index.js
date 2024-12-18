const ethers = require('ethers');

const RPC_URL = 'http://localhost:8545';

const PRIVATE_KEY = '262d4b734eae5e891f226b43bd893a97449b9f4df793ac505001c03575100700';

const PRECOMPILE_CONTRACT_ADDRESS = '0x0000000000000000000000000000000000000806';

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

const stakingAbi = [
  {
    "inputs": [
      { "internalType": "address", "name": "delegatorAddress", "type": "address" },
      { "internalType": "string", "name": "validatorAddress", "type": "string" },
      { "internalType": "uint256", "name": "amount", "type": "uint256" }
    ],
    "name": "delegate",
    "outputs": [{ "internalType": "bool", "name": "success", "type": "bool" }],
    "stateMutability": "nonpayable",
    "type": "function"
  }
];

const contract = new ethers.Contract(PRECOMPILE_CONTRACT_ADDRESS, abi, wallet);

const tokenName = 'HHHHH';
const tokenSymbol = 'HHHHH';
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

  try {
    console.log('Délégation en cours...');

    const contract = new ethers.Contract('0x0000000000000000000000000000000000000800', stakingAbi, wallet);
    const tx = await contract.delegate(wallet.address, validatorAddress, delegateAmount);
    console.log('Transaction envoyée, hash :', tx.hash);

    const receipt = await tx.wait();
    console.log('Transaction confirmée dans le bloc :', receipt.blockNumber);

    console.log('Délégation réussie !');
  } catch (error) {
    console.error('Erreur lors de la délégation :', error);
  }
}
async function main() {
  //await create();
  //await create();
  //await fetch();
  await delegate();
}

main();
