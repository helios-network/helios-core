const { ethers, deployments, getNamedAccounts } = require("hardhat");
const { toBech32 } = require("@cosmjs/encoding");
const { StargateClient } = require("@cosmjs/stargate");
const { assert } = require("chai");

describe("Counter Contract - Gas Consumption Analysis", function () {
	// Increase timeout for Helios blockchain transactions
	this.timeout(30000);

	let counterContract;
	let deployer;
	let deployerSigner;
	let evmAddress, heliosAddress;

	// Before each test, deploy the counter contract
	beforeEach(async function () {
		// Get the deployer account
		const accounts = await getNamedAccounts();
		deployer = accounts.deployer;
		deployerSigner = await ethers.getSigner(deployer);
		evmAddress = deployerSigner.address;
		heliosAddress = ethToHeliosAddress(evmAddress);

		await deployments.fixture(["Counter"]);
		const Counter = await deployments.get("Counter");
		counterContract = await ethers.getContractAt("Counter", Counter.address);

		// Log initial setup
		console.log(`Contract deployed at: ${counterContract.target}`);
		console.log(`Deployer EVM address: ${evmAddress}`);
		console.log(`Deployer Cosmos address: ${heliosAddress}`);
	});

	it("validate there are no discrepancies in gas consumption (EIP-1559)", async function () {
		let originalCount, newCount;
		originalCount = await counterContract.count();

		const originalEVMBalance = await ethers.provider.getBalance(evmAddress);
		const originalCosmosBalance = await getHeliosBalance(heliosAddress);
		assert.equal(originalCosmosBalance.amount, originalEVMBalance.toString(), "Initial EVM and Cosmos balances should be equal");
		
		const feeData = await ethers.provider.getFeeData();

		 // Send EIP-1559 transaction
		const tx = await counterContract.increment({
			maxFeePerGas: feeData.maxFeePerGas,
			maxPriorityFeePerGas: feeData.maxPriorityFeePerGas,
			type: 2
		});
		console.log(`Tx hash: ${tx.hash}`);

		const receipt = await tx.wait();
		// console.log(`Receipt: ${JSON.stringify(receipt, null, 2)}`);
		
		newCount = await counterContract.count();
		// assert that the counter was incremented by 1
		assert.equal(newCount, originalCount + 1n, "Counter should be incremented by 1");

		 // Get current balances
		const currentEVMBalance = await ethers.provider.getBalance(evmAddress);
		const currentCosmosBalance = await getHeliosBalance(heliosAddress)
		
		// Use gasPrice from receipt directly
		const gasPrice = receipt.gasPrice;
		const gasUsed = receipt.gasUsed;
		const actualGasCost = gasUsed * gasPrice;
		
		// Fetch block baseFeePerGas for additional verification
		const block = await ethers.provider.getBlock(receipt.blockNumber);

		// assert that the gas price is not greater than the max fee per gas
		assert.isAtMost(
			gasPrice,
			feeData.maxFeePerGas,
			"gasPrice should never exceed maxFeePerGas"
		);

		// assert that the gas price is at least equal to the block's baseFeePerGas
		assert.isAtLeast(
			gasPrice,
			block.baseFeePerGas,
			"gasPrice should be at least equal to block's baseFeePerGas"
		);

		// assert that the priority fee is not greater than the max priority fee per gas
		const actualPriorityFee = gasPrice - block.baseFeePerGas;
		assert.isAtMost(
			actualPriorityFee,
			feeData.maxPriorityFeePerGas,
			"Priority fee should never exceed maxPriorityFeePerGas"
		);

		// assert that the actual gas cost is not greater than the max fee per gas times the gas used
		const maxPossibleGasCost = gasUsed * feeData.maxFeePerGas;

		assert.isAtMost(
			actualGasCost,
			maxPossibleGasCost,
			"Actual gas cost should never exceed maxFeePerGas * gasUsed"
		);

		// Check balances match across layers after tx
		assert.equal(
			currentEVMBalance.toString(),
			currentCosmosBalance.amount,
			"Balances should match after transaction"
		);

		// Confirm the balance difference exactly matches gas cost
		const balanceDifference = originalEVMBalance - currentEVMBalance;
		assert.equal(
			balanceDifference.toString(),
			actualGasCost.toString(),
			"Balance difference should exactly equal actual gas cost"
		);
	});
});


function ethToHeliosAddress(ethAddress) {
	const addressBytes = ethers.getBytes(ethAddress);
	return toBech32("helios", addressBytes);
}

async function getHeliosBalance(heliosAddress) {
	const client = await StargateClient.connect(process.env.HELIOS_RPC_URL);
	const balance = await client.getBalance(heliosAddress, "ahelios");
	return balance;
}