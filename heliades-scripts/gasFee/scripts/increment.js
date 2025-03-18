const { ethers, deployments, getNamedAccounts } = require("hardhat");
async function main() {
	//const bridge = await (await ethers.getContractFactory("EVMBridgeConnectedToERC20Minter"))
	//   const CounterContract = await (await hre.ethers.getContractFactory("Counter")).attach("0x8cbF1A9167F66B9B3310Aab56E4fEFc17514d23A");
	const counter = await deployments.get("Counter");
	const CounterContract = await ethers.getContractAt("Counter", counter.address);

	const { deployer } = await getNamedAccounts();
	const originalBalance = await ethers.provider.getBalance(deployer);
	console.log(`deployer original balance: ${ethers.formatUnits(originalBalance, 'wei')} wei`);

	const deployerAddress = await ethers.getSigner(deployer);
	console.log(`deployer address: ${deployerAddress.address}`);

	// Preview what gas values would be used without sending the transaction
	const estimatedGas = await CounterContract.increment.estimateGas();
	console.log("Estimated gas:", estimatedGas.toString());

	// Get current gas price
	const currentFeeData = await ethers.provider.getFeeData();
	console.log("Current fee data:", JSON.stringify(currentFeeData, null, 2));

	const tx = await CounterContract.increment();

	console.log("tx hash: ", tx.hash);

	const receipt = await tx.wait();
	console.log("receipt: ", receipt);

	const count = await CounterContract.count();

	console.log('Count =', hre.ethers.formatUnits(count, 'wei'));


	const currentBalance = await ethers.provider.getBalance(deployer);
	console.log(`deployer new balance: ${ethers.formatUnits(currentBalance, 'wei')} wei`);

	// log gas used
	const gasUsed = receipt.gasUsed;
	const gasPrice = receipt.gasPrice;
	const totalGasCost = gasUsed * gasPrice;
	console.log(`gas used: ${gasUsed}, gas price: ${gasPrice}, total gas cost: ${totalGasCost} wei`);

	// calculate difference in balance
	const difference = currentBalance - originalBalance;
	console.log(`difference in balance: ${ethers.formatUnits(difference, 'wei')} wei`);

	// calculate difference in balance after gas cost
	const differenceAfterGasCost = difference + totalGasCost;
	console.log(`difference in balance after gas cost: ${ethers.formatUnits(differenceAfterGasCost, 'wei')} wei`);
}

main().catch((error) => {
	console.error(error);
	process.exitCode = 1;
});