const hre = require("hardhat");

async function main() {
  console.log("Deploying SimpleERC20Token contract...");
  
  // Get the deployer account
  const [deployer] = await hre.ethers.getSigners();
  console.log(`Deploying with account: ${deployer.address}`);
  
  // Get deployer balance
  const initialBalance = await hre.ethers.provider.getBalance(deployer.address);
  console.log(`Initial deployer balance: ${hre.ethers.formatEther(initialBalance)} HELIOS`);
  
  // Token parameters
  const tokenName = "Helios Test Token";
  const tokenSymbol = "HTT";
  const tokenDecimals = 18;
  const initialSupply = 1000000; // 1 million tokens
  
  // Deploy the contract
  const SimpleERC20Token = await hre.ethers.getContractFactory("SimpleERC20Token");
  
  console.log("Deploying contract...");
  const token = await SimpleERC20Token.deploy(
    tokenName,
    tokenSymbol,
    tokenDecimals,
    initialSupply
  );
  
  // Pour les versions récentes d'ethers.js
  const tx = token.deploymentTransaction();
  if (tx) {
    console.log(`Transaction hash: ${tx.hash}`);
  } else {
    console.log("No deployment transaction found");
  }
  
  console.log("Waiting for deployment confirmation...");
  
  await token.waitForDeployment();
  const tokenAddress = await token.getAddress();
  
  // Get final deployer balance
  const finalBalance = await hre.ethers.provider.getBalance(deployer.address);
  const gasCost = initialBalance - finalBalance;
  
  console.log(`\nDeployment successful!`);
  console.log(`Contract address: ${tokenAddress}`);
  console.log(`Token name: ${tokenName}`);
  console.log(`Token symbol: ${tokenSymbol}`);
  console.log(`Token decimals: ${tokenDecimals}`);
  console.log(`Initial supply: ${initialSupply} ${tokenSymbol}`);
  console.log(`Gas cost: ${hre.ethers.formatEther(gasCost)} HELIOS`);
  
  console.log(`\nTo use this token in other scripts, run:`);
  console.log(`export TOKEN_ADDRESS=${tokenAddress}`);
  
  console.log(`\nCheck your Helios node logs for:`);
  console.log(`- "Detected new ERC20 token" messages`);
  console.log(`- "erc20_detected" events`);
}

main()
  .then(() => process.exit(0))
  .catch((error) => {
    console.error(error);
    process.exit(1);
  });