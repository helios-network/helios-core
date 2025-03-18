const hre = require("hardhat");
const {ethers, deployments} = require("hardhat");

async function main() {

  const counter = await deployments.get("Counter");
  const CounterContract = await ethers.getContractAt("Counter", counter.address);

  
  const count = await CounterContract.count();

  console.log('Count =', hre.ethers.formatUnits(count, 'wei'));
}

main().catch((error) => {
  console.error(error);
  process.exitCode = 1;
});