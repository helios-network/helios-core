require("@nomicfoundation/hardhat-ethers"); // For latest Hardhat versions
// require("@nomiclabs/hardhat-ethers"); // Use this for older Hardhat versions

/** @type import('hardhat/config').HardhatUserConfig */
module.exports = {
  solidity: {
    compilers: [
      { version: "0.8.20" },
      { version: "0.4.22" },
    ],
  },
  paths: {
    sources: "./solidity",
  },
  networks: {
    localhost: {
      url: "http://127.0.0.1:8545",
    },
  },
};
