const { ethers } = require("ethers");
const fs = require('fs');
const { simpleWebSocketSubscription } = require('./event-subscription');

const EVM_RPC_URL='http://localhost:8545'
const GENESIS_PRIVATE_KEY1='2c37c3d09d7a1c957f01ad200cec69bc287d0a9cc85b4dce694611a4c9c24036'
const GENESIS_PRIVATE_KEY2='a7b40e40feaf9492097aeaac24c5ae3a2c32bef35961d67a0a6b263e9a9a69c1'

async function sleep(ms) {
    return new Promise(resolve => setTimeout(resolve, ms));
}

async function main() {
    console.log("\n=== Auto-Registration Test Started ===\n");
    
    const provider = new ethers.JsonRpcProvider(EVM_RPC_URL);

    // Subscribe to events
    let subscription;
    try {
        subscription = simpleWebSocketSubscription();
    } catch (error) {
        console.error("Error subscribing to events:", error);
        process.exitCode = 1;
        return;
    }
    
    // Setup deployer and user accounts
    const deployer = new ethers.Wallet(GENESIS_PRIVATE_KEY1, provider);
    const user = new ethers.Wallet(GENESIS_PRIVATE_KEY2, provider);

    // Transfer some HELIOS to the user for testing
    console.log("\nTransferring 20 HELIOS to the user...");
    const tx = await deployer.sendTransaction({
        to: user.address,
        value: ethers.parseUnits("20", "ether")
    });
    await tx.wait();
    console.log("Transferred 20 HELIOS to the user!");

    // Deploy a new contract
    console.log("\nDeploying new contract for testing...");
    const counterArtifact = JSON.parse(
        fs.readFileSync('artifacts/contracts/Counter.sol/Counter.json', 'utf8')
    );
    
    const factory = new ethers.ContractFactory(
        counterArtifact.abi,
        counterArtifact.bytecode,
        deployer
    );

    const counter = await factory.deploy({
        gasLimit: 500000,
        maxFeePerGas: ethers.parseUnits("500", "gwei"),
        maxPriorityFeePerGas: ethers.parseUnits("100", "gwei"),
        type: 2,
        chainId: 4242
    });

    console.log("Waiting for contract deployment...");
    await counter.waitForDeployment();
    const contractAddress = await counter.getAddress();
    console.log("Contract deployed at:", contractAddress);

    // Wait for auto-registration
    console.log("\nWaiting for auto-registration to complete...");
    await sleep(5000);

    // Get initial balances
    const deployerInitialBalance = await provider.getBalance(deployer.address);
    const userInitialBalance = await provider.getBalance(user.address);
    
    console.log("\nInitial Balances:");
    console.log("Deployer:", ethers.formatEther(deployerInitialBalance), "HELIOS");
    console.log("User:", ethers.formatEther(userInitialBalance), "HELIOS");
    
    // Execute transactions to test revenue distribution
    console.log("\n=== Testing Revenue Distribution ===");
    const txCount = 3;
    const receipts = [];
    
    let nonce = await provider.getTransactionCount(user.address);
    console.log("Starting nonce:", nonce);

    for(let i = 0; i < txCount; i++) {
        console.log(`\nSending transaction ${i + 1}/${txCount}...`);
        
        try {
            const tx = await counter.connect(user).increment({
                gasLimit: 500000,
                maxFeePerGas: ethers.parseUnits("500", "gwei"),
                maxPriorityFeePerGas: ethers.parseUnits("100", "gwei"),
                type: 2,
                nonce: nonce++
            });
            
            console.log("Transaction hash:", tx.hash);
            const receipt = await tx.wait();
            console.log("Transaction mined in block:", receipt.blockNumber);
            receipts.push(receipt);
            await sleep(2000);
            
        } catch (error) {
            console.error("Transaction failed:", error.message);
            break;
        }
    }

    // Calculate results
    const totalGasUsed = receipts.reduce((sum, r) => sum + r.gasUsed, BigInt(0));
    const totalFees = receipts.reduce((sum, r) => sum + (r.gasUsed * r.gasPrice), BigInt(0));
    
    const deployerFinalBalance = await provider.getBalance(deployer.address);
    const userFinalBalance = await provider.getBalance(user.address);
    
    console.log("\n=== Results ===");
    console.log("Total Gas Used:", totalGasUsed.toString());
    console.log("Total Fees Paid:", ethers.formatEther(totalFees), "HELIOS");
    
    console.log("\nBalance Changes:");
    console.log("Deployer:");
    console.log("  Initial:", ethers.formatEther(deployerInitialBalance), "HELIOS");
    console.log("  Final:", ethers.formatEther(deployerFinalBalance), "HELIOS");
    console.log("  Change:", ethers.formatEther(deployerFinalBalance - deployerInitialBalance), "HELIOS");
    
    console.log("\nUser:");
    console.log("  Initial:", ethers.formatEther(userInitialBalance), "HELIOS");
    console.log("  Final:", ethers.formatEther(userFinalBalance), "HELIOS");
    console.log("  Change:", ethers.formatEther(userFinalBalance - userInitialBalance), "HELIOS");
    
    const expectedRevenue = totalFees * BigInt(10) / BigInt(100);
    console.log("\nExpected Revenue (10% of fees):", ethers.formatEther(expectedRevenue), "HELIOS");
    
    const actualRevenue = deployerFinalBalance - deployerInitialBalance;
    console.log("Actual Revenue:", ethers.formatEther(actualRevenue), "HELIOS");
    
    const percentageReceived = (actualRevenue * BigInt(100)) / totalFees;
    console.log("Percentage of Fees Received:", percentageReceived.toString(), "%");

    // Close event subscription
    subscription.close();
    console.log("\n=== Auto-Registration Test Completed ===\n");
}

main().catch((error) => {
    console.error(error);
    process.exitCode = 1;
}); 