const hre = require("hardhat");

async function main() {
  const CounterContract = await hre.ethers.getContractFactory("Counter");
  const smt = await CounterContract.deploy();

  console.log(smt);

  const response = await smt.waitForDeployment();

  console.log("response", response);

  console.log(
    `Counter deployed to ${smt.target}`
  );
}

// We recommend this pattern to be able to use async/await everywhere
// and properly handle errors.
main().catch((error) => {
  console.error(error);
  process.exitCode = 1;
});