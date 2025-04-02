const { ethers } = require("ethers");
require('dotenv').config({ path: '../.env' });
const fs = require('fs');
const { toBech32 } = require("@cosmjs/encoding");

const EVM_RPC_URL='http://localhost:8545'
const GENESIS_PRIVATE_KEY1='2c37c3d09d7a1c957f01ad200cec69bc287d0a9cc85b4dce694611a4c9c24036'

async function main() {
    console.log("\n=== Deployment Script Started ===\n");

    // Setup provider and wallets
    const provider = new ethers.JsonRpcProvider(EVM_RPC_URL);
    const deployer = new ethers.Wallet(GENESIS_PRIVATE_KEY1, provider);

	// Log deployer information
	console.log("Deployer Information:");
	console.log("  EVM Address (hex):", deployer.address);
	console.log("  Helios Address (bech32):", ethToHeliosAddress(deployer.address));
	console.log("  Balance:", ethers.formatEther(await provider.getBalance(deployer.address)), "HELIOS");

    // Read Counter contract artifact
    const counterArtifact = JSON.parse(
        fs.readFileSync('artifacts/contracts/Counter.sol/Counter.json', 'utf8')
    );
    
    try {
        console.log("\nDeploying Counter Contract...");
        
        // Get the nonce for the deployment transaction
        const nonce = await provider.getTransactionCount(deployer.address);
		console.log("  Deployment Nonce:", nonce);
        // Create contract factory
        const factory = new ethers.ContractFactory(
            counterArtifact.abi,
            counterArtifact.bytecode,
            deployer
        );

        // Deploy with specific transaction parameters
        const counter = await factory.deploy({
            gasLimit: 500000,
            maxFeePerGas: ethers.parseUnits("500", "gwei"),
            maxPriorityFeePerGas: ethers.parseUnits("100", "gwei"),
            type: 2,
            chainId: 4242,
            nonce: nonce
        });

        console.log("  Transaction hash:", counter.deploymentTransaction().hash);
        console.log("  Waiting for deployment confirmation...");
        
        await counter.waitForDeployment();
        
        console.log("\nCounter Contract Deployed!");
        console.log("  Contract Address:", await counter.getAddress());
		
        
        console.log("\n=== Deployment Script Completed ===\n");
    } catch (error) {
        console.error('Detailed error:', {
            error: error.message,
            data: error.data,
            transaction: error.transaction,
            code: error.code,
            reason: error.reason,
            raw: error
        });
        throw error;
    }
}

// Helper function to convert EVM address to Helios bech32 address
function ethToHeliosAddress(ethAddress) {
    const addressBytes = ethers.getBytes(ethAddress);
    return toBech32("helios", addressBytes);
}

// Handle errors in main
main().catch((error) => {
    console.error(error);
    process.exitCode = 1;
});