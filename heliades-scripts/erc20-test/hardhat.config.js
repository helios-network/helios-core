require("@nomicfoundation/hardhat-toolbox");
require('dotenv').config();

const GENESIS_PRIVATE_KEY = process.env.GENESIS_PRIVATE_KEY || "2c37c3d09d7a1c957f01ad200cec69bc287d0a9cc85b4dce694611a4c9c24036";

/** @type import('hardhat/config').HardhatUserConfig */
module.exports = {
  solidity: "0.8.28", // Changer cette version pour correspondre à celle de vos contrats
  networks: {
    helios: {
      url: "http://localhost:8545",
      chainId: 4242,
      accounts: [`0x${GENESIS_PRIVATE_KEY}`],
      gasPrice: 500000000000, // 500 gwei
      gas: 2100000
    }
  }
};