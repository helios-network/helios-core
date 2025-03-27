require("@nomicfoundation/hardhat-toolbox");
require("hardhat-deploy");
require("hardhat-gas-reporter");
require('dotenv').config();


/** @type import('hardhat/config').HardhatUserConfig */
module.exports = {
  solidity: "0.8.28",
  networks: {
    helios: {
      url: "http://localhost:8545",
      chainId: 4242,
      accounts: [process.env.GENESIS_PRIVATE_KEY1],
    },
	hardhat: {
		chainId: 31337,
	},
  },
  namedAccounts: {
    deployer: {
		  default: 0,
			helios: 0,
    },
  },
  gasReporter: {
    enabled: true,
    currency: 'USD',
    gasPrice: 1010000000,
    noColors: false,
    outputFile: "gas-report.txt",
    showMethodSig: true,
	token: "HELIOS",
	showTimeSpent: true,
  },
};

