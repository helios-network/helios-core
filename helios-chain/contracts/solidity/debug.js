const { ethers } = require("hardhat");

async function main() {
  const provider = new ethers.JsonRpcProvider("http://127.0.0.1:8545"); // Local Hardhat node
  const signer = await provider.getSigner(); // Get the first signer (default account)
  
  const contractAddress = "0x1D54EcB8583Ca25895c512A8308389fFD581F9c9";
  const contract = await ethers.getContractAt("ERC20MinterBurnerDecimals", contractAddress, signer); // Attach signer

  const result = await contract.symbol(); // Call function
  console.log("Result:", result);
}

main().catch((error) => {
  console.error(error);
  process.exitCode = 1;
});
