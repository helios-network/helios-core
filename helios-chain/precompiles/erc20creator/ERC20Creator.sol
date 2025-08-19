// SPDX-License-Identifier: LGPL-3.0-only
pragma solidity >=0.8.17;

/// @dev The address of the ERC20Creator precompiled contract.
address constant ERC20Creator_PRECOMPILE_ADDRESS = 0x0000000000000000000000000000000000000806;

/// @dev An instance of the ERC20Creator precompiled contract.
ERC20Creator constant ERC20CREATOR_CONTRACT = ERC20Creator(ERC20Creator_PRECOMPILE_ADDRESS);

interface ERC20Creator {
    /**
     * @dev Creates a new ERC20 token with the specified parameters and role authorities.
     * Returns the address of the newly created ERC20 token.
     * @param name The name of the ERC20 token.
     * @param symbol The symbol of the ERC20 token.
     * @param denom The denomination of one unit of the ERC20 token.
     * @param totalSupply The total supply of the ERC20 token.
     * @param decimals The number of decimals of the ERC20 token.
     * @param logoBase64 The logo in base64 png 200x200 optional "".
     * @param mintAuthority Who can mint tokens (address(0) = non-mintable).
     * @param pauseAuthority Who can pause/unpause transfers (address(0) = non-pausable).
     * @param burnAuthority Who can burn tokens (address(0) = no special burn authority).
     * @return tokenAddress The address of the newly created ERC20 token.
     */
    function createErc20(
        string memory name,
        string memory symbol,
        string memory denom,
        uint256 totalSupply,
        uint8 decimals,
        string memory logoBase64,
        address mintAuthority,
        address pauseAuthority,
        address burnAuthority
    ) external returns (address tokenAddress);
}