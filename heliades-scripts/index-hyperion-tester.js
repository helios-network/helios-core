const ethers = require('ethers');
const WebSocket = require('ws');
const fs = require('fs');

const RPC_HELIOS_URL = 'https://testnet1.helioschainlabs.org';
const RPC_BSC_TESTNET_URL = '';
const PRIVATE_KEY = '';
const provider = new ethers.JsonRpcProvider(RPC_HELIOS_URL);
const providerBscTestnet = new ethers.JsonRpcProvider(RPC_BSC_TESTNET_URL);
const walletConnectedToHelios = new ethers.Wallet(PRIVATE_KEY, provider);
const walletConnectedToBscTestnet = new ethers.Wallet(PRIVATE_KEY, providerBscTestnet);

async function sendToChain(amount) {
  try {
    console.log('sendToChain en cours...');

    const hyperionAbi = JSON.parse(fs.readFileSync('../helios-chain/precompiles/hyperion/abi.json').toString()).abi;
    const contract = new ethers.Contract('0x0000000000000000000000000000000000000900', hyperionAbi, walletConnectedToHelios);
    const tx = await contract.sendToChain( // I'm validator and
      97, // chainId
      "0x688feDf2cc9957eeD5A56905b1A0D74a3bAc0000", // receiver
      "0xd567B3d7B8FE3C79a1AD8dA978812cfC4Fa05e75", // token address (ex: USDT)
      ethers.parseEther(amount), // amount to transfer (ex: 10 USDT)
      ethers.parseEther("1"), // fee you want to pay (ex: 1 USDT)
      {
        gasLimit: 500000,
        gasPrice: ethers.parseUnits('2', "gwei")
      }
    );
    console.log('Transaction envoyée, hash :', tx.hash);

    const receipt = await tx.wait();
    console.log('Transaction confirmée dans le bloc :', receipt.blockNumber);
    console.log(receipt);
  } catch (error) {
    console.error('Erreur lors de la sendToChain :', error);
  }
}

async function sendToHelios(amount) {
  try {
    console.log('sendToHelios en cours...');

    // approve token on BSC testnet
    const tokenAbi = [
      'function approve(address spender, uint256 amount) external returns (bool)',
    ];
    const tokenContract = new ethers.Contract('0xC689BF5a007F44410676109f8aa8E3562da1c9Ba', tokenAbi, walletConnectedToBscTestnet);
    const approveTx = await tokenContract.approve("0x4afDe6B4141A6c2Bc7c164846578F635d418DaB3", ethers.parseEther(amount));
    console.log('Approve transaction envoyée, hash :', approveTx.hash);
    const approveReceipt = await approveTx.wait();
    console.log('Approve transaction confirmée dans le bloc :', approveReceipt.blockNumber);

    const hyperionAbi = [
      {
        "inputs": [
          {
            "internalType": "address",
            "name": "_tokenContract",
            "type": "address"
          },
          {
            "internalType": "bytes32",
            "name": "_destination",
            "type": "bytes32"
          },
          {
            "internalType": "uint256",
            "name": "_amount",
            "type": "uint256"
          },
          {
            "internalType": "string",
            "name": "_data",
            "type": "string"
          }
        ],
        "name": "sendToHelios",
        "outputs": [],
        "stateMutability": "nonpayable",
        "type": "function"
      }
    ];
    const contract = new ethers.Contract('0x4afDe6B4141A6c2Bc7c164846578F635d418DaB3', hyperionAbi, walletConnectedToBscTestnet);
    
    const destinationBytes32 = ethers.zeroPadValue(
      "0x688feDf2cc9957eeD5A56905b1A0D74a3bAc0000",
      32
    );
    const tx = await contract.sendToHelios( // I'm validator and
      "0xC689BF5a007F44410676109f8aa8E3562da1c9Ba",
      destinationBytes32,
      ethers.parseEther(amount), // amount to transfer (ex: 10 USDT)
      "",
      {
        gasLimit: 500000,
        gasPrice: ethers.parseUnits('0.2', "gwei")
      }
    );
    console.log('Transaction envoyée, hash :', tx.hash);

    const receipt = await tx.wait();
    console.log('Transaction confirmée dans le bloc :', receipt.blockNumber);
    console.log(receipt);
  } catch (error) {
    console.error('Erreur lors de la sendToHelios :', error);
  }
}

async function getTokenBalance(chainId, tokenAddress) {
  const abi = [
    'function balanceOf(address account) view returns (uint256)',
  ];
  if (chainId === 97) {
    const contract = new ethers.Contract(tokenAddress, abi, providerBscTestnet);
    const balance = await contract.balanceOf(walletConnectedToBscTestnet.address);
    console.log("balance of WBNB on BSC testnet :", ethers.formatEther(balance));
    return ethers.formatEther(balance);
  } else if (chainId === 42000) {
    const contract = new ethers.Contract(tokenAddress, abi, provider);
    const balance = await contract.balanceOf(walletConnectedToHelios.address);
    return ethers.formatEther(balance);
  } else {
    throw new Error("Invalid chainId");
  }
}

async function transferTokenTo(tokenAddress, amount, destinationAddress) {
  const abi = [
    'function transfer(address to, uint256 amount) external returns (bool)',
    'function decimals() view returns (uint8)',
  ];
  try {
    console.log('transferTokenTo en cours...');
    const contract = new ethers.Contract(tokenAddress, abi, walletConnectedToHelios);
    const decimals = await contract.decimals();
    const amountInWei = ethers.parseUnits(amount, decimals);
    const tx = await contract.transfer(destinationAddress, amountInWei);
    console.log('Transaction envoyée, hash :', tx.hash);
    const receipt = await tx.wait();
    console.log('Transaction confirmée dans le bloc :', receipt.blockNumber);
    console.log(receipt);
  } catch (error) {
    console.error('Erreur lors de la transferTokenTo :', error);
  }
}

async function main() {

  await transferTokenTo("0x9c88f904f24253a7d5fbb30b5a2094d0ce7678dc", "0.1", "0x9ec7c2AA72C26903d681322Ff01f2AFea2E5Ab60");

  // // get balance of WBNB on each chains
  // let newWbnbBalanceOnHeliosChain = 0;
  // let newWbnbBalanceOnBscTestnet = 0;
  // let wbnbBalanceOnBscTestnet = await getTokenBalance(97, "0xC689BF5a007F44410676109f8aa8E3562da1c9Ba");
  // let wbnbBalanceOnHeliosChain = await getTokenBalance(42000, "0xd567B3d7B8FE3C79a1AD8dA978812cfC4Fa05e75");
  // console.log("wbnbBalanceOnBscTestnet :", wbnbBalanceOnBscTestnet);
  // console.log("wbnbBalanceOnHeliosChain :", wbnbBalanceOnHeliosChain);
  
  // // // 1 send 0.001 from bsc testnet to helios chain
  // await sendToHelios("0.001");

  // let startTime = Date.now();

  // // // waiting check on helios chain our balance of WBNB
  // while (true) {
  //   newWbnbBalanceOnHeliosChain = await getTokenBalance(42000, "0xd567B3d7B8FE3C79a1AD8dA978812cfC4Fa05e75");
  //   if (newWbnbBalanceOnHeliosChain > wbnbBalanceOnHeliosChain) {
  //     console.log("WBNB balance on Helios chain increased to :", newWbnbBalanceOnHeliosChain);
  //     console.log("Transaction successful!");
  //     break;
  //   }
  //   console.log("waiting for WBNB balance on Helios chain to increase...");
  //   await new Promise(resolve => setTimeout(resolve, 5000));
  // }
  // let endTime = Date.now();
  // console.log("Time taken :", (endTime - startTime) / 1000, "seconds");
  // console.log("WBNB balance on Helios chain increased to :", newWbnbBalanceOnHeliosChain);

  // wbnbBalanceOnBscTestnet = await getTokenBalance(97, "0xC689BF5a007F44410676109f8aa8E3562da1c9Ba");
  // // 2 send 0.001 from helios chain to bsc testnet
  // await sendToChain("0.001");
  // let startBackwardTime = Date.now();
  // // waiting check on bsc testnet our balance of WBNB
  // while (true) {
  //   newWbnbBalanceOnBscTestnet = await getTokenBalance(97, "0xC689BF5a007F44410676109f8aa8E3562da1c9Ba");
  //   if (newWbnbBalanceOnBscTestnet > wbnbBalanceOnBscTestnet) {
  //     console.log("WBNB balance on BSC testnet increased to :", newWbnbBalanceOnBscTestnet);
  //     console.log("Transaction successful!");
  //     break;
  //   }
  //   console.log("waiting for WBNB balance on BSC testnet to increase...");
  //   await new Promise(resolve => setTimeout(resolve, 5000));
  // }
  // let endBackwardTime = Date.now();
  // console.log("Time taken :", (endBackwardTime - startBackwardTime) / 1000, "seconds");
  // console.log("WBNB balance on BSC testnet increased to :", newWbnbBalanceOnBscTestnet);

  // console.log("All transactions successful!");
  // // stats:
  // console.log("Stats:");
  // console.log("Total Time taken :", ((endTime - startTime) / 1000) + ((endBackwardTime - startBackwardTime) / 1000), "seconds");
  // console.log("Time taken for deposit transaction :", (endTime - startTime) / 1000, "seconds");
  // console.log("Time taken for withdraw transaction :", (endBackwardTime - startBackwardTime) / 1000, "seconds");
}

main();