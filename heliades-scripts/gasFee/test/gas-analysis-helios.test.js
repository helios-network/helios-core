const { ethers, deployments, getNamedAccounts } = require("hardhat");
const { toBech32 } = require("@cosmjs/encoding");
const { StargateClient } = require("@cosmjs/stargate");
const { assert, expect } = require("chai");
const fs = require("fs");
const path = require('path');

describe("Gas Consumption Analysis Across Network Interactions", function () {
	// Increase timeout for Helios blockchain transactions
	this.timeout(30000);

	let deployer, deployerSigner;
	let evmAddress, heliosAddress;

	// Setup before tests
	before(async function () {
		const accounts = await getNamedAccounts();
		deployer = accounts.deployer;
		deployerSigner = await ethers.getSigner(deployer);
		evmAddress = deployerSigner.address;
		heliosAddress = ethToHeliosAddress(evmAddress);
	});

	describe("Counter Contract Interaction", function() {
		let counterContract, originalCount, newCount;
		let txReceipt;
		
		before(async function() {
			// Deploy counter contract
			await deployments.fixture(["Counter"]);
			const Counter = await deployments.get("Counter");
			counterContract = await ethers.getContractAt("Counter", Counter.address);
			
			// Get initial state
			originalCount = await counterContract.count();
			
			// Get initial balances and fee data
			const originalEVMBalance = await ethers.provider.getBalance(evmAddress);
			const originalCosmosBalance = await getHeliosBalance(heliosAddress);
			const feeData = await ethers.provider.getFeeData();
			
			// Execute the transaction
			const tx = await counterContract.increment({
				maxFeePerGas: feeData.maxFeePerGas,
				maxPriorityFeePerGas: feeData.maxPriorityFeePerGas,
				type: 2
			});
			
			// Get receipt
			txReceipt = await tx.wait();
			
			// Get the new state
			newCount = await counterContract.count();
			
			// Get current balances
			const currentEVMBalance = await ethers.provider.getBalance(evmAddress);
			const currentCosmosBalance = await getHeliosBalance(heliosAddress);
			
			// Store data for tests
			this.testData = {
				originalEVMBalance,
				originalCosmosBalance,
				currentEVMBalance,
				currentCosmosBalance,
				receipt: txReceipt,
				feeData,
				gasUsed: txReceipt.gasUsed,
				gasPrice: txReceipt.gasPrice,
				actualGasCost: txReceipt.gasUsed * txReceipt.gasPrice,
				block: await ethers.provider.getBlock(txReceipt.blockNumber)
			};
			
			// Log gas metrics
			console.log(`\n--- Smart Contract Interaction Gas Metrics ---`);
			console.log(`Gas Used: ${this.testData.gasUsed.toString()}`);
			console.log(`Gas Price: ${this.testData.gasPrice.toString()}`);
			console.log(`Actual Gas Cost: ${this.testData.actualGasCost.toString()}`);
			console.log(`Base Fee Per Gas: ${this.testData.block.baseFeePerGas.toString()}`);
		});
		
		// Business logic test
		it("should increment the counter by 1", function() {
			assert.equal(
				newCount,
				originalCount + 1n,
				"Counter should be incremented by 1"
			);
		});
		
		// Gas analysis tests
		it("should have matching initial EVM and Cosmos balances", function() {
			assert.equal(
				this.testData.originalCosmosBalance.amount,
				this.testData.originalEVMBalance.toString(),
				`Initial EVM and Cosmos balances should be equal`
			);
		});
		
		it("should never exceed maxFeePerGas for gasPrice", function() {
			assert.isAtMost(
				this.testData.gasPrice,
				this.testData.feeData.maxFeePerGas,
				"gasPrice should never exceed maxFeePerGas"
			);
		});
		
		it("should have gasPrice at least equal to block's baseFeePerGas", function() {
			assert.isAtLeast(
				this.testData.gasPrice,
				this.testData.block.baseFeePerGas,
				"gasPrice should be at least equal to block's baseFeePerGas"
			);
		});
		
		it("should have priority fee within expected bounds", function() {
			const actualPriorityFee = this.testData.gasPrice - this.testData.block.baseFeePerGas;
			assert.isAtMost(
				actualPriorityFee,
				this.testData.feeData.maxPriorityFeePerGas,
				"Priority fee should never exceed maxPriorityFeePerGas"
			);
		});
		
		it("should have actual gas cost within expected bounds", function() {
			const maxPossibleGasCost = this.testData.gasUsed * this.testData.feeData.maxFeePerGas;
			assert.isAtMost(
				this.testData.actualGasCost,
				maxPossibleGasCost,
				"Actual gas cost should never exceed maxFeePerGas * gasUsed"
			);
		});
		
		it("should have matching EVM and Cosmos balances after transaction", function() {
			assert.equal(
				this.testData.currentEVMBalance.toString(),
				this.testData.currentCosmosBalance.amount,
				"Balances should match after transaction"
			);
		});
		
		it("should have balance difference equal to gas cost", function() {
			const balanceDifference = this.testData.originalEVMBalance - this.testData.currentEVMBalance;
			const expectedDifference = BigInt(this.testData.actualGasCost.toString());
			assert.equal(
				balanceDifference.toString(),
				expectedDifference.toString(),
				"Balance difference should equal gas cost"
			);
		});
	});

	describe("Token Transfer", function() {
		let testRecipient, transferAmount;
		let recipientOriginalBalance, recipientNewBalance;
		
		before(async function() {
			// Generate a random recipient address for transfer tests
			const randomWallet = ethers.Wallet.createRandom();
			testRecipient = randomWallet.address;
			transferAmount = ethers.parseEther("0.01");
			
			console.log(`Test recipient address: ${testRecipient}`);
			
			// Get initial state
			recipientOriginalBalance = await ethers.provider.getBalance(testRecipient);
			
			// Get initial balances and fee data
			const originalEVMBalance = await ethers.provider.getBalance(evmAddress);
			const originalCosmosBalance = await getHeliosBalance(heliosAddress);
			const feeData = await ethers.provider.getFeeData();
			
			// Execute the transaction
			const tx = await deployerSigner.sendTransaction({
				to: testRecipient,
				value: transferAmount,
				maxFeePerGas: feeData.maxFeePerGas,
				maxPriorityFeePerGas: feeData.maxPriorityFeePerGas,
				type: 2
			});
			
			// Get receipt
			const txReceipt = await tx.wait();
			
			// Get new state
			recipientNewBalance = await ethers.provider.getBalance(testRecipient);
			
			// Get current balances
			const currentEVMBalance = await ethers.provider.getBalance(evmAddress);
			const currentCosmosBalance = await getHeliosBalance(heliosAddress);
			
			// Store data for tests
			this.testData = {
				originalEVMBalance,
				originalCosmosBalance,
				currentEVMBalance,
				currentCosmosBalance,
				receipt: txReceipt,
				feeData,
				gasUsed: txReceipt.gasUsed,
				gasPrice: txReceipt.gasPrice,
				actualGasCost: txReceipt.gasUsed * txReceipt.gasPrice,
				block: await ethers.provider.getBlock(txReceipt.blockNumber),
				transferAmount
			};
			
			// Log gas metrics
			console.log(`\n--- Token Transfer Gas Metrics ---`);
			console.log(`Gas Used: ${this.testData.gasUsed.toString()}`);
			console.log(`Gas Price: ${this.testData.gasPrice.toString()}`);
			console.log(`Actual Gas Cost: ${this.testData.actualGasCost.toString()}`);
			console.log(`Base Fee Per Gas: ${this.testData.block.baseFeePerGas.toString()}`);
		});
		
		// Business logic test
		it("should transfer exact amount to recipient", function() {
			assert.equal(
				recipientNewBalance - recipientOriginalBalance,
				transferAmount,
				"Recipient should receive exact transfer amount"
			);
		});
		
		// Gas analysis tests
		it("should have matching initial EVM and Cosmos balances", function() {
			assert.equal(
				this.testData.originalCosmosBalance.amount,
				this.testData.originalEVMBalance.toString(),
				`Initial EVM and Cosmos balances should be equal`
			);
		});
		
		it("should never exceed maxFeePerGas for gasPrice", function() {
			assert.isAtMost(
				this.testData.gasPrice,
				this.testData.feeData.maxFeePerGas,
				"gasPrice should never exceed maxFeePerGas"
			);
		});
		
		it("should have gasPrice at least equal to block's baseFeePerGas", function() {
			assert.isAtLeast(
				this.testData.gasPrice,
				this.testData.block.baseFeePerGas,
				"gasPrice should be at least equal to block's baseFeePerGas"
			);
		});
		
		it("should have priority fee within expected bounds", function() {
			const actualPriorityFee = this.testData.gasPrice - this.testData.block.baseFeePerGas;
			assert.isAtMost(
				actualPriorityFee,
				this.testData.feeData.maxPriorityFeePerGas,
				"Priority fee should never exceed maxPriorityFeePerGas"
			);
		});
		
		it("should have actual gas cost within expected bounds", function() {
			const maxPossibleGasCost = this.testData.gasUsed * this.testData.feeData.maxFeePerGas;
			assert.isAtMost(
				this.testData.actualGasCost,
				maxPossibleGasCost,
				"Actual gas cost should never exceed maxFeePerGas * gasUsed"
			);
		});
		
		it("should have matching EVM and Cosmos balances after transaction", function() {
			assert.equal(
				this.testData.currentEVMBalance.toString(),
				this.testData.currentCosmosBalance.amount,
				"Balances should match after transaction"
			);
		});
		
		it("should have balance difference equal to gas cost plus transfer amount", function() {
			const balanceDifference = this.testData.originalEVMBalance - this.testData.currentEVMBalance;
			const expectedDifference = BigInt(this.testData.actualGasCost.toString()) + BigInt(this.testData.transferAmount.toString());
			assert.equal(
				balanceDifference.toString(),
				expectedDifference.toString(),
				"Balance difference should equal gas cost plus transfer amount"
			);
		});
	});

	describe("Precompile Interaction", function() {
		let govPrecompileContract;
		const initialDeposit = "1000000000000000000";
		
		before(async function() {
			const govPrecompileAddress = "0x0000000000000000000000000000000000000805";
			const abiPath = path.resolve(__dirname, '../../../heliades-scripts/gasFee/contracts/precompile/gov_abi.json');
			const govAbi = JSON.parse(
				fs.readFileSync(abiPath, "utf8")
			);
			govPrecompileContract = new ethers.Contract(govPrecompileAddress, govAbi, deployerSigner);
			
			// Get initial balances and fee data
			const originalEVMBalance = await ethers.provider.getBalance(evmAddress);
			const originalCosmosBalance = await getHeliosBalance(heliosAddress);
			const feeData = await ethers.provider.getFeeData();
			
			// Define transaction parameters
			const title = 'Whitelist WETH into the consensus with a base stake of power 100';
			const description = 'Explaining why WETH would be a good potential for Helios consensus and why it would secure the market';
			const assets = [
				{
					denom: 'WETH',
					contractAddress: '0x80b5a32E4F032B2a058b4F29EC95EEfEEB87aDcd', 
					chainId: 'ethereum',                                          
					decimals: 6,
					baseWeight: 100,
					metadata: 'WETH stablecoin'
				}
			];
			
			// Execute the transaction
			const tx = await govPrecompileContract.addNewAssetProposal(
				title,
				description,
				assets,
				initialDeposit,
				{
					maxFeePerGas: feeData.maxFeePerGas,
					maxPriorityFeePerGas: feeData.maxPriorityFeePerGas,
					type: 2
				}
			);
			
			// Get receipt
			const txReceipt = await tx.wait();
			
			// Get current balances
			const currentEVMBalance = await ethers.provider.getBalance(evmAddress);
			const currentCosmosBalance = await getHeliosBalance(heliosAddress);
			
			// Store data for tests
			this.testData = {
				originalEVMBalance,
				originalCosmosBalance,
				currentEVMBalance,
				currentCosmosBalance,
				receipt: txReceipt,
				feeData,
				gasUsed: txReceipt.gasUsed,
				gasPrice: txReceipt.gasPrice,
				actualGasCost: txReceipt.gasUsed * txReceipt.gasPrice,
				block: await ethers.provider.getBlock(txReceipt.blockNumber),
				initialDeposit
			};
			
			// Log gas metrics
			console.log(`\n--- Precompile Interaction Gas Metrics ---`);
			console.log(`Gas Used: ${this.testData.gasUsed.toString()}`);
			console.log(`Gas Price: ${this.testData.gasPrice.toString()}`);
			console.log(`Actual Gas Cost: ${this.testData.actualGasCost.toString()}`);
			console.log(`Base Fee Per Gas: ${this.testData.block.baseFeePerGas.toString()}`);
		});
		
		// Gas analysis tests
		it("should have matching initial EVM and Cosmos balances", function() {
			assert.equal(
				this.testData.originalCosmosBalance.amount,
				this.testData.originalEVMBalance.toString(),
				`Initial EVM and Cosmos balances should be equal`
			);
		});
		
		it("should never exceed maxFeePerGas for gasPrice", function() {
			assert.isAtMost(
				this.testData.gasPrice,
				this.testData.feeData.maxFeePerGas,
				"gasPrice should never exceed maxFeePerGas"
			);
		});
		
		it("should have gasPrice at least equal to block's baseFeePerGas", function() {
			assert.isAtLeast(
				this.testData.gasPrice,
				this.testData.block.baseFeePerGas,
				"gasPrice should be at least equal to block's baseFeePerGas"
			);
		});
		
		it("should have priority fee within expected bounds", function() {
			const actualPriorityFee = this.testData.gasPrice - this.testData.block.baseFeePerGas;
			assert.isAtMost(
				actualPriorityFee,
				this.testData.feeData.maxPriorityFeePerGas,
				"Priority fee should never exceed maxPriorityFeePerGas"
			);
		});
		
		it("should have actual gas cost within expected bounds", function() {
			const maxPossibleGasCost = this.testData.gasUsed * this.testData.feeData.maxFeePerGas;
			assert.isAtMost(
				this.testData.actualGasCost,
				maxPossibleGasCost,
				"Actual gas cost should never exceed maxFeePerGas * gasUsed"
			);
		});
		
		it("should have matching EVM and Cosmos balances after transaction", function() {
			assert.equal(
				this.testData.currentEVMBalance.toString(),
				this.testData.currentCosmosBalance.amount,
				"Balances should match after transaction"
			);
		});
		
		it("should have balance difference equal to gas cost plus deposit amount", function() {
			const balanceDifference = this.testData.originalEVMBalance - this.testData.currentEVMBalance;
			const expectedDifference = BigInt(this.testData.actualGasCost.toString()) + BigInt(initialDeposit);
			assert.equal(
				balanceDifference.toString(),
				expectedDifference.toString(),
				"Balance difference should equal gas cost plus deposit amount"
			);
		});
	});
});

// Helper functions
function ethToHeliosAddress(ethAddress) {
	const addressBytes = ethers.getBytes(ethAddress);
	return toBech32("helios", addressBytes);
}

async function getHeliosBalance(heliosAddress) {
	const client = await StargateClient.connect(process.env.HELIOS_RPC_URL);
	const balance = await client.getBalance(heliosAddress, "ahelios");
	return balance;
}
