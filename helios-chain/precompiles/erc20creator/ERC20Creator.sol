// SPDX-License-Identifier: LGPL-3.0-only
pragma solidity >=0.8.17;

/// @dev The address of the ERC20Creator precompiled contract.
address constant ERC20Creator_PRECOMPILE_ADDRESS = 0x0000000000000000000000000000000000000806;

/// @dev An instance of the ERC20Creator precompiled contract.
ERC20Creator constant ERC20CREATOR_CONTRACT = ERC20Creator(ERC20Creator_PRECOMPILE_ADDRESS);

interface ERC20Creator {
    /**
     * @dev Creates a new ERC20 token with the specified parameters.
     * Returns the address of the newly created ERC20 token.
     * @param name The name of the ERC20 token.
     * @param symbol The symbol of the ERC20 token.
     * @param totalSupply The total supply of the ERC20 token.
     * @param decimals The number of decimals of the ERC20 token.
     * @return tokenAddress The address of the newly created ERC20 token.
     */
    function createErc20(
        string memory name,
        string memory symbol,
        uint256 totalSupply,
        uint8 decimals
    ) external returns (address tokenAddress);
}
